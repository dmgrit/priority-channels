package priority_channels

import "context"

type lock struct {
	l chan struct{}
}

func newLock() *lock {
	return &lock{l: make(chan struct{}, 1)}
}

func (l *lock) Lock() {
	l.l <- struct{}{}
}

func (l *lock) Unlock() {
	<-l.l
}

func (l *lock) TryLockWithContext(ctx context.Context) bool {
	select {
	case l.l <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}
