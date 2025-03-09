package synchronization

import "context"

func TryLockOrExitOnState(lock *Lock, stateTracker *RepeatingStateTracker) bool {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if stateTracker.Await(ctx) {
			cancel()
		}
	}()
	if !lock.TryLockWithContext(ctx) {
		return false
	}
	cancel()
	return true
}
