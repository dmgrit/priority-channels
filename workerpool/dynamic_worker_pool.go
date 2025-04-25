package workerpool

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/dmgrit/priority-channels/internal/synchronization"
)

type token struct{}

type DynamicWorkerPool[T any] struct {
	processFn                      func(T)
	activeGoroutines               atomic.Int32
	started                        atomic.Uint32
	state                          stateManager
	ctx                            context.Context
	cancel                         context.CancelFunc
	done                           chan struct{}
	restartFromStoppedStateTracker *synchronization.RepeatingStateTracker
}

func NewDynamicWorkerPool[T any](ctx context.Context, processFn func(T), numWorkers int) (*DynamicWorkerPool[T], error) {
	if numWorkers < 0 {
		return nil, errors.New("number of workers must be a non-negative number")
	}

	origSem := make(chan token, numWorkers)
	stateCh := make(chan poolState, 1)
	initState := poolState{
		sem:            origSem,
		desiredWorkers: numWorkers,
	}
	stateCh <- initState

	ctxWithCancel, cancel := context.WithCancel(ctx)
	return &DynamicWorkerPool[T]{
		state:                          stateCh,
		processFn:                      processFn,
		done:                           make(chan struct{}),
		ctx:                            ctxWithCancel,
		cancel:                         cancel,
		restartFromStoppedStateTracker: synchronization.NewRepeatingStateTracker(),
	}, nil
}

func (p *DynamicWorkerPool[T]) Process(msgs <-chan T) error {
	if !p.started.CompareAndSwap(0, 1) {
		return errors.New("worker pool already started")
	}
	go func() {
		defer close(p.done)
		for {
			select {
			case <-p.ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				sem, currWorkersNum := p.state.Get()
				if currWorkersNum == 0 {
					// Process the last read message so it isn't dropped,
					// then block and wait for an explicit signal that workers have restarted.
					p.activeGoroutines.Add(1)
					p.processFn(msg)
					p.activeGoroutines.Add(-1)
					if !p.restartFromStoppedStateTracker.Await(p.ctx) {
						return
					}
					continue
				}
				sem <- token{}
				go func(sem chan token) {
					p.activeGoroutines.Add(1)
					p.processFn(msg)
					p.activeGoroutines.Add(-1)
					<-sem
				}(sem)
			}
		}
	}()
	return nil
}

func (p *DynamicWorkerPool[T]) Shutdown() {
	p.cancel()
}

func (p *DynamicWorkerPool[T]) Done() <-chan struct{} {
	return p.done
}

func (p *DynamicWorkerPool[T]) UpdateWorkersNum(newWorkersNum int) error {
	if newWorkersNum < 0 {
		return errors.New("number of workers must be a non-negative number")
	}
	desiredWorkers := newWorkersNum

	alreadyAligned, updateAlreadyInProgress := p.state.TryStartWorkersUpdate(desiredWorkers)
	if alreadyAligned || updateAlreadyInProgress {
		return nil
	}
	go func() {
		var finished bool
		for !finished {
			select {
			case <-p.ctx.Done():
				finished = true
			default:
				p.doUpdateWorkersNum(desiredWorkers)
				finished, desiredWorkers = p.state.FinishWorkersUpdateIfAligned()
			}
		}
	}()

	return nil
}

func (p *DynamicWorkerPool[T]) doUpdateWorkersNum(newWorkersNum int) {
	updateRes := p.state.UpdateSemaphore(func(prevSem chan token) chan token {
		currWorkersNum := cap(prevSem)
		newSem := make(chan token, newWorkersNum)
		for i := 0; i < currWorkersNum && i < newWorkersNum; i++ {
			newSem <- token{}
		}
		return newSem
	})
	currWorkersNum := updateRes.prevWorkers
	prevSem := updateRes.prevSem
	newSem := updateRes.sem

	if newWorkersNum == 0 && currWorkersNum > 0 {
		// we stopped all workers, reset the tracker to be able to listen on start event again
		p.restartFromStoppedStateTracker.Reset()
	} else if newWorkersNum > 0 && currWorkersNum == 0 {
		p.restartFromStoppedStateTracker.Broadcast()
	}

	// wait for all currently running tasks to finish
	for i := currWorkersNum; i > 0; i-- {
		prevSem <- token{}
		if i <= newWorkersNum {
			<-newSem
		}
	}
}

func (p *DynamicWorkerPool[T]) WorkersNum() int {
	_, numWorkers := p.state.Get()
	return numWorkers
}

func (p *DynamicWorkerPool[T]) ActiveWorkersNum() int {
	return int(p.activeGoroutines.Load())
}

type poolState struct {
	sem              chan token
	desiredWorkers   int
	updateInProgress bool
}

type stateManager chan poolState

func (m stateManager) Get() (sem chan token, numWorkers int) {
	state := <-m
	sem = state.sem
	numWorkers = cap(sem)
	m <- state
	return sem, numWorkers
}

func (m stateManager) TryStartWorkersUpdate(desiredWorkersNum int) (alreadyAligned bool, updateAlreadyInProgress bool) {
	state := <-m
	state.desiredWorkers = desiredWorkersNum
	if state.desiredWorkers == cap(state.sem) {
		alreadyAligned = true
	}
	updateAlreadyInProgress = state.updateInProgress
	if !state.updateInProgress {
		state.updateInProgress = true
	}
	m <- state
	return alreadyAligned, updateAlreadyInProgress
}

func (m stateManager) FinishWorkersUpdateIfAligned() (bool, int) {
	var res bool
	var desiredWorkers int
	state := <-m
	desiredWorkers = state.desiredWorkers
	if state.desiredWorkers == cap(state.sem) {
		state.updateInProgress = false
		res = true
	}
	m <- state
	return res, desiredWorkers
}

func (m stateManager) UpdateSemaphore(updateSem func(chan token) chan token) *updateStateResult {
	res := &updateStateResult{}
	state := <-m
	if updateSem != nil {
		res.prevSem = state.sem
		res.prevWorkers = cap(state.sem)
		state.sem = updateSem(res.prevSem)
	}
	res.sem = state.sem
	res.numWorkers = cap(state.sem)
	m <- state
	return res
}

type updateStateResult struct {
	sem         chan token
	numWorkers  int
	prevSem     chan token
	prevWorkers int
}
