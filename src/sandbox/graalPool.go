package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/ol/common"
)

// GraalPool is a ContainerFactory that creates graal containers.
type GraalPool struct {
}

// NewGraalPool creates a GraalPool.
func NewGraalPool(_ string, _ []string) (*GraalPool, error) {
	fmt.Println("Creating a GraalPool")
	// cmd := exec.Command("")

	// stdout, err := cmd.Output()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }
	// fmt.Print(string(stdout))

	pool := &GraalPool{}

	return pool, nil
}

// Create creates a graal sandbox from the handler and sandbox directory.
func (pool *GraalPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta, _rtType common.RuntimeType) (sb Sandbox, err error) {
	c := &GraalContainer{
		hostDir: scratchDir,
	}

	// wrap to make thread-safe and handle container death
	safe := newSafeSandbox(c)
	// safe.startNotifyingListeners(pool.eventHandlers)
	return safe, nil
}

// Cleanup will free up any unneeded data/resources
// Currently, this function does nothing and cleanup is handled by the graal daemon
func (pool *GraalPool) Cleanup() {}

// DebugString returns debug information
func (pool *GraalPool) DebugString() string {
	return "pool.debugger.Dump()"
}

// AddListener allows registering event handlers
func (pool *GraalPool) AddListener(handler SandboxEventFunc) {
}
