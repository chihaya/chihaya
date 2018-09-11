// Package stop implements a pattern for shutting down a group of processes.
package stop

import (
	"sync"
)

// Channel is used to return zero or more errors asynchronously. Call Done()
// once to pass errors to the Channel.
type Channel chan []error

// Result is a receive-only version of Channel. Call Wait() once to receive any
// returned errors.
type Result <-chan []error

// Done adds zero or more errors to the Channel and closes it, indicating the
// caller has finished stopping. It should be called exactly once.
func (ch Channel) Done(errs ...error) {
	if len(errs) > 0 && errs[0] != nil {
		ch <- errs
	}
	close(ch)
}

// Result converts a Channel to a Result.
func (ch Channel) Result() <-chan []error {
	return ch
}

// Wait blocks until Done() is called on the underlying Channel and returns any
// errors. It should be called exactly once.
func (r Result) Wait() []error {
	return <-r
}

// AlreadyStopped is a closed error channel to be used by Funcs when
// an element was already stopped.
var AlreadyStopped Result

// AlreadyStoppedFunc is a Func that returns AlreadyStopped.
var AlreadyStoppedFunc = func() Result { return AlreadyStopped }

func init() {
	closeMe := make(Channel)
	close(closeMe)
	AlreadyStopped = closeMe.Result()
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
	Stop() Result
}

// Func is a function that can be used to provide a clean shutdown.
type Func func() Result

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
func (cg *Group) Stop() Result {
	cg.Lock()
	defer cg.Unlock()

	whenDone := make(Channel)

	waitChannels := make([]Result, 0, len(cg.stoppables))
	for _, toStop := range cg.stoppables {
		waitFor := toStop()
		if waitFor == nil {
			panic("received a nil chan from Stop")
		}
		waitChannels = append(waitChannels, waitFor)
	}

	go func() {
		var errors []error
		for _, waitForMe := range waitChannels {
			childErrors := waitForMe.Wait()
			if len(childErrors) > 0 {
				errors = append(errors, childErrors...)
			}
		}
		whenDone.Done(errors...)
	}()

	return whenDone.Result()
}
