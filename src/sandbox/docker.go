/*

Provides the mechanism for managing a given Docker container-based lambda.

Must be paired with a DockerContainerManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/ol/common"
)

// DockerContainer is a sandbox inside a docker container.
type DockerContainer struct {
	hostID	string
	hostDir   string
	nspid	 string
	container *docker.Container
	client	*docker.Client
	installed map[string]bool
	meta	  *SandboxMeta
	rtType   common.RuntimeType
}

type HandlerState int

const (
	Unitialized HandlerState = iota
	Running
	Paused
)

func (h HandlerState) String() string {
	switch h {
	case Unitialized:
		return "unitialized"
	case Running:
		return "running"
	case Paused:
		return "paused"
	default:
		panic("Unknown state!")
	}
}

// dockerError adds details (sandbox log, state, etc.) to an error.
func (c *DockerContainer) dockerError(outer error) (err error) {
	buf := bytes.NewBufferString(outer.Error() + ".  ")

	if err := c.InspectUpdate(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not inspect container (%v).  ", err.Error()))
	} else {
		buf.WriteString(fmt.Sprintf("Container state is <%v>.  ", c.container.State.StateString()))
	}

	if log, err := c.Logs(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not fetch [%s] logs!\n", c.container.ID))
	} else {
		buf.WriteString(fmt.Sprintf("<--- Start handler container [%s] logs: --->\n", c.container.ID))
		buf.WriteString(log)
		buf.WriteString(fmt.Sprintf("<--- End handler container [%s] logs --->\n", c.container.ID))
	}

	return errors.New(buf.String())
}

// InspectUpdate calls docker inspect to update the state of the container.
func (c *DockerContainer) InspectUpdate() error {
	container, err := c.client.InspectContainer(c.container.ID)
	if err != nil {
		return err
	}
	c.container = container

	return nil
}

// State returns the state of the Docker sandbox.
func (c *DockerContainer) State() (hstate HandlerState, err error) {
	if err := c.InspectUpdate(); err != nil {
		return hstate, err
	}

	if c.container.State.Running {
		if c.container.State.Paused {
			hstate = Paused
		} else {
			hstate = Running
		}
	} else {
		return hstate, fmt.Errorf("unexpected state")
	}

	return hstate, nil
}

// HTTPProxy returns a file socket channel for direct communication with the sandbox.
func (c *DockerContainer) HTTPProxy() (p *httputil.ReverseProxy, err error) {
	sockPath := filepath.Join(c.hostDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}

	tr := &http.Transport{Dial: dial}
	u, err := url.Parse("http://sock-container")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = tr
	return proxy, nil
}

// Start starts the container.
func (c *DockerContainer) start() error {
	if err := c.client.StartContainer(c.container.ID, nil); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return c.dockerError(err)
	}

	container, err := c.client.InspectContainer(c.container.ID)
	if err != nil {
		log.Printf("failed to inpect container with err %v\n", err)
		return c.dockerError(err)
	}
	c.container = container
	c.nspid = fmt.Sprintf("%d", container.State.Pid)

	return nil
}

// Pause stops/freezes the container.
func (c *DockerContainer) Pause() error {
	st, err := c.State()
	if err != nil {
		return err
	} else if st == Paused {
		return nil
	}

	if err := c.client.PauseContainer(c.container.ID); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return c.dockerError(err)
	}

	return nil
}

// Unpause resumes/unfreezes the container.
func (c *DockerContainer) Unpause() error {
	st, err := c.State()
	if err != nil {
		return err
	} else if st == Running {
		return nil
	}

	if err := c.client.UnpauseContainer(c.container.ID); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", c.container.Name, err)
		return c.dockerError(err)
	}

	return nil
}

// Destroy shuts down this container
func (c *DockerContainer) Destroy() {
	if err := c.internalDestroy(); err != nil {
		panic(fmt.Sprintf("Failed to cleanup container %v: %v", c.container.ID, err))
	}
}

// frees all resources associated with the lambda
func (c *DockerContainer) internalDestroy() error {
	c.Unpause()

	// TODO(tyler): is there any advantage to trying to stop
	// before killing?  (i.e., use SIGTERM instead SIGKILL)
	opts := docker.KillContainerOptions{ID: c.container.ID}
	if err := c.client.KillContainer(opts); err != nil {
		log.Printf("failed to kill container with error %v\n", err)
		return c.dockerError(err)
	}

	// remove sockets if they exist
	if err := os.RemoveAll(filepath.Join(c.hostDir, "ol.sock")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(c.hostDir, "fs.sock")); err != nil {
		return err
	}

	if err := c.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: c.container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return c.dockerError(err)
	}

	return nil
}

// Logs returns log output for the container.
func (c *DockerContainer) Logs() (string, error) {
	stdoutPath := filepath.Join(c.hostDir, "stdout")
	stderrPath := filepath.Join(c.hostDir, "stderr")

	stdout, err := ioutil.ReadFile(stdoutPath)
	if err != nil {
		return "", err
	}

	stderr, err := ioutil.ReadFile(stderrPath)
	if err != nil {
		return "", err
	}

	stdoutHdr := fmt.Sprintf("Container (%s) stdout:", c.container.ID)
	stderrHdr := fmt.Sprintf("Container (%s) stderr:", c.container.ID)
	ret := fmt.Sprintf("%s\n%s\n%s\n%s\n", stdoutHdr, stdout, stderrHdr, stderr)

	return ret, nil
}

// GetRuntimeLog returns the log of the runtime
// Note, this is not supported for docker yet
func (c *DockerContainer) GetRuntimeLog() string {
	return "" //TODO
}

// GetProxyLog returns the log of the http proxy
// Note, this is not supported for docker yet
func (c *DockerContainer) GetProxyLog() string {
	return "" //TODO
}

// NSPid returns the pid of the first process of the docker container.
func (c *DockerContainer) NSPid() string {
	return c.nspid
}

// ID returns the identifier of this container
func (c *DockerContainer) ID() string {
	return c.hostID
}

// GetRuntimeType returns what runtime is being used by this container?
func (c *DockerContainer) GetRuntimeType() common.RuntimeType {
	return c.rtType
}

// DockerID returns the id assigned by docker itself, not by open lambda
func (c *DockerContainer) DockerID() string {
	return c.container.ID
}

// HostDir returns the host directory of this container
func (c *DockerContainer) HostDir() string {
	return c.hostDir
}

func (c *DockerContainer) runServer() error {
	if c.rtType != common.RT_PYTHON {
		return fmt.Errorf("Unsupported runtime")
	}

	cmd := []string{"python3", "/runtimes/python/server_legacy.py"}

	execOpts := docker.CreateExecOptions{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Container:    c.container.ID,
		Cmd:          cmd,
	}

	if exec, err := c.client.CreateExec(execOpts); err != nil {
		return err
	} else if err := c.client.StartExec(exec.ID, docker.StartExecOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *DockerContainer) DebugString() string {
	return fmt.Sprintf("SANDBOX %s (DOCKER)\n", c.ID())
}

func (c *DockerContainer) fork(dst Sandbox) (err error) {
	panic("DockerContainer does not implement cross-container forks")
}

func (c *DockerContainer) childExit(child Sandbox) {
	panic("DockerContainers should not have children because fork is unsupported")
}

func waitForServerPipeReady(hostDir string) error {
	// upon success, the goroutine will send nil; else, it will send the error
	ready := make(chan error, 1)

	go func() {
		pipeFile := filepath.Join(hostDir, "server_pipe")
		pipe, err := os.OpenFile(pipeFile, os.O_RDWR, 0777)
		if err != nil {
			log.Printf("Cannot open pipe: %v\n", err)
			return
		}
		defer pipe.Close()

		// wait for "ready"
		buf := make([]byte, 5)
		_, err = pipe.Read(buf)
		if err != nil {
			ready <- fmt.Errorf("cannot read from stdout of sandbox :: %v", err)
		} else if string(buf) != "ready" {
			ready <- fmt.Errorf("expect to see `ready` but got %s", string(buf))
		}
		ready <- nil
	}()

	// TODO: make timeout configurable
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()

	select {
	case err := <-ready:
		return err
	case <-timeout.C:
		return fmt.Errorf("instance server failed to initialize after 20s")
	}
}

func (c *DockerContainer) Status(key SandboxStatus) (string, error) {
	return "", STATUS_UNSUPPORTED
}

func (c *DockerContainer) Meta() *SandboxMeta {
	return c.meta
}
