package priority_channels

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmgrit/priority-channels/internal/synchronization"
)

type token struct{}

type DynamicWorkerPool[T any] struct {
	priorityChannel                *PriorityChannel[T]
	activeGoroutines               atomic.Int32
	started                        atomic.Uint32
	state                          stateManager
	pendingReceiveContextCancel    context.CancelFunc
	pendingReceiveContextCancelMtx sync.Mutex
	ctx                            context.Context
	cancel                         context.CancelFunc
	done                           chan struct{}
	restartFromStoppedStateTracker *synchronization.RepeatingStateTracker
	priorityChannelUpdatesMtx      sync.Mutex
	isStopped                      bool
	exitReason                     ExitReason
	exitReasonChannelName          string
}

func NewDynamicWorkerPool[T any](ctx context.Context, priorityChannel *PriorityChannel[T], numWorkers int) (*DynamicWorkerPool[T], error) {
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
		priorityChannel:                priorityChannel,
		state:                          stateCh,
		done:                           make(chan struct{}),
		ctx:                            ctxWithCancel,
		cancel:                         cancel,
		restartFromStoppedStateTracker: synchronization.NewRepeatingStateTracker(),
	}, nil
}

func (p *DynamicWorkerPool[T]) Process(processFn func(Delivery[T])) error {
	fnGetResult := func(msg T, details ReceiveDetails) Delivery[T] {
		return Delivery[T]{Msg: msg, ReceiveDetails: details}
	}
	return doProcess(p, fnGetResult, processFn)
}

func (p *DynamicWorkerPool[T]) ProcessMessages(processFn func(T)) error {
	fnGetResult := func(msg T, details ReceiveDetails) T {
		return msg
	}
	return doProcess(p, fnGetResult, processFn)
}

func doProcess[T any, R any](p *DynamicWorkerPool[T], fnGetResult func(msg T, details ReceiveDetails) R, processFn func(R)) error {
	if !p.started.CompareAndSwap(0, 1) {
		return errors.New("worker pool already started")
	}
	go func() {
		defer close(p.done)
		for {
			select {
			case <-p.ctx.Done():
				return
			default:
				sem, currWorkersNum := p.state.Get()
				if currWorkersNum == 0 {
					if !p.restartFromStoppedStateTracker.Await(p.ctx) {
						return
					}
					continue
				}
				sem <- token{}
				ctx, cancel := context.WithCancel(p.ctx)
				p.setPendingReceiveContextCancel(cancel)
				msg, receiveDetails, status := p.priorityChannel.ReceiveWithContextEx(ctx)
				if status == ReceiveSuccess {
					go func(sem chan token) {
						p.activeGoroutines.Add(1)
						processFn(fnGetResult(msg, receiveDetails))
						p.activeGoroutines.Add(-1)
						<-sem
					}(sem)
					continue
				} else if status == ReceivePriorityChannelClosed && receiveDetails.ChannelName == "" {
					return
				} else if status != ReceiveContextCancelled {
					p.setPaused(status.ExitReason(), receiveDetails.ChannelName)
					if status == ReceiveChannelClosed || status == ReceivePriorityChannelClosed {
						var channelType ChannelType
						if status == ReceiveChannelClosed {
							channelType = InputChannelType
						} else {
							channelType = PriorityChannelType
						}
						if p.priorityChannel.AwaitRecover(context.Background(), receiveDetails.ChannelName, channelType) {
							p.setResumed()
						}
					} else {
						time.Sleep(time.Microsecond * 100)
					}
				}
				<-sem
			}
		}
	}()
	return nil
}

func (p *DynamicWorkerPool[T]) setPendingReceiveContextCancel(cancel context.CancelFunc) {
	p.pendingReceiveContextCancelMtx.Lock()
	p.pendingReceiveContextCancel = cancel
	p.pendingReceiveContextCancelMtx.Unlock()
}

func (p *DynamicWorkerPool[T]) callPendingReceiveContextCancel() {
	p.pendingReceiveContextCancelMtx.Lock()
	if p.pendingReceiveContextCancel != nil {
		p.pendingReceiveContextCancel()
	}
	p.pendingReceiveContextCancelMtx.Unlock()
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
	p.callPendingReceiveContextCancel()
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

// Status returns whether the consumer is stopped, and if so, the reason for stopping and,
// in case the reason is a closed channel, the name of the channel that was closed.
func (p *DynamicWorkerPool[T]) Status() (stopped bool, reason ExitReason, channelName string) {
	p.priorityChannelUpdatesMtx.Lock()
	defer p.priorityChannelUpdatesMtx.Unlock()

	select {
	case <-p.Done():
		return true, PriorityChannelClosed, ""
	default:
		return p.isStopped, p.exitReason, p.exitReasonChannelName
	}
}

func (p *DynamicWorkerPool[T]) setPaused(exitReason ExitReason, exitReasonChannelName string) {
	p.priorityChannelUpdatesMtx.Lock()
	defer p.priorityChannelUpdatesMtx.Unlock()

	p.isStopped = true
	p.exitReason = exitReason
	p.exitReasonChannelName = exitReasonChannelName
}

func (p *DynamicWorkerPool[T]) setResumed() {
	p.priorityChannelUpdatesMtx.Lock()
	defer p.priorityChannelUpdatesMtx.Unlock()

	p.isStopped = false
	p.exitReason = UnknownExitReason
	p.exitReasonChannelName = ""
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
