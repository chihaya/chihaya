// Package stop implements a pattern for shutting down a group of processes.
package stop

import (
	"sync"
)

// AlreadyStopped is a closed error channel to be used by Funcs when
// an element was already stopped.
var AlreadyStopped <-chan error

// AlreadyStoppedFunc is a Func that returns AlreadyStopped.
var AlreadyStoppedFunc = func() <-chan error { return AlreadyStopped }

func init() {
	closeMe := make(chan error)
	close(closeMe)
	AlreadyStopped = closeMe
}

// Stopper is an interface that allows a clean shutdown.
type Stopper interface {
	// Stop returns a channel that indicates whether the stop was
	// successful.
	//
	// The channel can either return one error or be closed.
	// Closing the channel signals a clean shutdown.
	// Stop() should return immediately and perform the actual shutdown in a
	// separate goroutine.
	Stop() <-chan error
}

// Func is a function that can be used to provide a clean shutdown.
type Func func() <-chan error

// Group is a collection of Stoppers that can be stopped all at once.
type Group struct {
	stoppables []Func
	sync.Mutex
}

// NewGroup allocates a new Group.
func NewGroup() *Group {
	return &Group{
		stoppables: make([]Func, 0),
	}
}

// Add appends a Stopper to the Group.
func (cg *Group) Add(toAdd Stopper) {
	cg.Lock()
	defer cg.Unlock()

	cg.stoppables = append(cg.stoppables, toAdd.Stop)
}

// AddFunc appends a Func to the Group.
func (cg *Group) AddFunc(toAddFunc Func) {
	cg.Lock()
	defer cg.Unlock()

	cg.stoppables = append(cg.stoppables, toAddFunc)
}

// Stop stops all members of the Group.
//
// Stopping will be done in a concurrent fashion.
// The slice of errors returned contains all errors returned by stopping the
// members.
func (cg *Group) Stop() []error {
	cg.Lock()
	defer cg.Unlock()

	var errors []error
	whenDone := make(chan struct{})

	waitChannels := make([]<-chan error, 0, len(cg.stoppables))
	for _, toStop := range cg.stoppables {
		waitFor := toStop()
		if waitFor == nil {
			panic("received a nil chan from Stop")
		}
		waitChannels = append(waitChannels, waitFor)
	}

	go func() {
		for _, waitForMe := range waitChannels {
			err := <-waitForMe
			if err != nil {
				errors = append(errors, err)
			}
		}
		close(whenDone)
	}()

	<-whenDone
	return errors
}
