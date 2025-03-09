package synchronization

import "context"

// Adapted from the excellent presentation below - 'Communication: Repeating Transition' pattern
// [Bryan Mills's talk on concurrency patterns]: https://drive.google.com/file/d/1nPdvhB0PutEJzdCq5ms6UI58dp50fcAN/view
type RepeatingStateTracker struct {
	next chan chan struct{}
}

func NewRepeatingStateTracker() *RepeatingStateTracker {
	next := make(chan chan struct{}, 1)
	// in the beginning event has not occurred yet
	// so initializing with state for waiting for it to happen
	next <- make(chan struct{})
	return &RepeatingStateTracker{next: next}
}

func (a *RepeatingStateTracker) Await(ctx context.Context) bool {
	nextState := <-a.next
	a.next <- nextState
	if nextState != nil {
		select {
		case <-ctx.Done():
			return false
		case <-nextState:
		}
	}
	return true
}

func (a *RepeatingStateTracker) Broadcast() {
	state := <-a.next
	if state != nil {
		// signal that event happened to all goroutines waiting on it
		close(state)
		state = nil
	}
	a.next <- state
}

func (a *RepeatingStateTracker) Reset() {
	state := <-a.next
	if state == nil {
		// prepare state for signalling when it will happen
		state = make(chan struct{})
	}
	a.next <- state
}
