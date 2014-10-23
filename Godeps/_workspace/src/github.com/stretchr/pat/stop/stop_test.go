package stop_test

import (
	"testing"
	"time"

	"github.com/stretchr/pat/stop"
)

type testStopper struct {
	stopChan chan stop.Signal
}

func NewTestStopper() *testStopper {
	s := new(testStopper)
	s.stopChan = stop.Make()
	return s
}

func (t *testStopper) Stop(wait time.Duration) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(t.stopChan)
	}()
}
func (t *testStopper) StopChan() <-chan stop.Signal {
	return t.stopChan
}

type noopStopper struct{}

func (t *noopStopper) Stop() {
}
func (t *noopStopper) StopChan() <-chan stop.Signal {
	return stop.Stopped()
}

func TestStop(t *testing.T) {

	s := NewTestStopper()
	s.Stop(1 * time.Second)
	stopChan := s.StopChan()
	select {
	case <-stopChan:
	case <-time.After(1 * time.Second):
		t.Error("Stop signal was never sent (timed out)")
	}

}

func TestAll(t *testing.T) {

	s1 := NewTestStopper()
	s2 := NewTestStopper()
	s3 := NewTestStopper()

	select {
	case <-stop.All(1*time.Second, s1, s2, s3):
	case <-time.After(1 * time.Second):
		t.Error("All signal was never sent (timed out)")
	}

}

func TestNoop(t *testing.T) {

	s := new(noopStopper)
	s.Stop()
	stopChan := s.StopChan()
	select {
	case <-stopChan:
	case <-time.After(1 * time.Second):
		t.Error("Stop signal was never sent (timed out)")
	}

}
