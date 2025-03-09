package synchronization

import "context"

type Lock struct {
	l chan struct{}
}

func NewLock() *Lock {
	return &Lock{l: make(chan struct{}, 1)}
}

func (l *Lock) Lock() {
	l.l <- struct{}{}
}

func (l *Lock) Unlock() {
	<-l.l
}

func (l *Lock) TryLockWithContext(ctx context.Context) bool {
	select {
	case l.l <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}
