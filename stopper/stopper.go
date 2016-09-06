package stopper

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
	// The channel can either return one error or be closed. Closing the
	// channel signals a clean shutdown.
	// The Stop function should return immediately and perform the actual
	// shutdown in a separate goroutine.
	Stop() <-chan error
}

// StopGroup is a group that can be stopped.
type StopGroup struct {
	stoppables     []Func
	stoppablesLock sync.Mutex
}

// Func is a function that can be used to provide a clean shutdown.
type Func func() <-chan error

// NewStopGroup creates a new StopGroup.
func NewStopGroup() *StopGroup {
	return &StopGroup{
		stoppables: make([]Func, 0),
	}
}

// Add adds a Stopper to the StopGroup.
// On the next call to Stop(), the Stopper will be stopped.
func (cg *StopGroup) Add(toAdd Stopper) {
	cg.stoppablesLock.Lock()
	defer cg.stoppablesLock.Unlock()

	cg.stoppables = append(cg.stoppables, toAdd.Stop)
}

// AddFunc adds a Func to the StopGroup.
// On the next call to Stop(), the Func will be called.
func (cg *StopGroup) AddFunc(toAddFunc Func) {
	cg.stoppablesLock.Lock()
	defer cg.stoppablesLock.Unlock()

	cg.stoppables = append(cg.stoppables, toAddFunc)
}

// Stop stops all members of the StopGroup.
// Stopping will be done in a concurrent fashion.
// The slice of errors returned contains all errors returned by stopping the
// members.
func (cg *StopGroup) Stop() []error {
	cg.stoppablesLock.Lock()
	defer cg.stoppablesLock.Unlock()

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
