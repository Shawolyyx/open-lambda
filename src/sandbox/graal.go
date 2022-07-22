/*

Provides the mechanism for managing a given Docker container-based lambda.

Must be paired with a GraalContainerManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"fmt"
	"net/http/httputil"
	"net/url"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/common"
)

// GraalContainer is a sandbox inside a docker container.
type GraalContainer struct {
	hostDir string
}

// InspectUpdate calls docker inspect to update the state of the container.
func (c *GraalContainer) InspectUpdate() error {
	return nil
}

// HTTPProxy returns a file socket channel for direct communication with the sandbox.
func (c *GraalContainer) HTTPProxy() (p *httputil.ReverseProxy, err error) {
	sockPath := filepath.Join(c.hostDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	u, err := url.Parse("http://localhost:8080")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	return proxy, nil
}

// Start starts the container.
func (c *GraalContainer) start() error {
	return nil
}

// Pause stops/freezes the container.
func (c *GraalContainer) Pause() error {
	return nil
}

// Unpause resumes/unfreezes the container.
func (c *GraalContainer) Unpause() error {
	return nil
}

// Destroy shuts down this container
func (c *GraalContainer) Destroy() {
}

// frees all resources associated with the lambda
func (c *GraalContainer) internalDestroy() error {
	return nil
}

// Logs returns log output for the container.
func (c *GraalContainer) Logs() (string, error) {
	return "", nil
}

// GetRuntimeLog returns the log of the runtime
// Note, this is not supported for docker yet
func (c *GraalContainer) GetRuntimeLog() string {
	return "" //TODO
}

// GetProxyLog returns the log of the http proxy
// Note, this is not supported for docker yet
func (c *GraalContainer) GetProxyLog() string {
	return "" //TODO
}

// NSPid returns the pid of the first process of the docker container.
func (c *GraalContainer) NSPid() string {
	return "c.nspid"
}

// ID returns the identifier of this container
func (c *GraalContainer) ID() string {
	return "c.hostID"
}

// GetRuntimeType returns what runtime is being used by this container?
func (c *GraalContainer) GetRuntimeType() common.RuntimeType {
	return common.RT_NATIVE
}

// DockerID returns the id assigned by docker itself, not by open lambda
func (c *GraalContainer) DockerID() string {
	return "c.container.ID"
}

// HostDir returns the host directory of this container
func (c *GraalContainer) HostDir() string {
	return c.hostDir
}

func (c *GraalContainer) runServer() error {

	return nil
}

func (c *GraalContainer) DebugString() string {
	return fmt.Sprintf("SANDBOX %s (DOCKER)\n", c.ID())
}

func (c *GraalContainer) fork(dst Sandbox) (err error) {
	panic("GraalContainer does not implement cross-container forks")
}

func (c *GraalContainer) childExit(child Sandbox) {
	panic("GraalContainers should not have children because fork is unsupported")
}

func (c *GraalContainer) Status(key SandboxStatus) (string, error) {
	return "", STATUS_UNSUPPORTED
}

func (c *GraalContainer) Meta() *SandboxMeta {
	return nil
}
