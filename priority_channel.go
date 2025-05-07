package priority_channels

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/dmgrit/priority-channels/internal/selectable"
	psync "github.com/dmgrit/priority-channels/internal/synchronization"
)

type PriorityChannel[T any] struct {
	ctx                         context.Context
	ctxCancel                   context.CancelFunc
	controlC                    chan controlMessage[T]
	compositeChannel            selectable.Channel[T]
	channelReceiveWaitInterval  *time.Duration
	lock                        *psync.Lock
	blockWaitAllChannelsTracker *psync.RepeatingStateTracker
	notifyMtx                   sync.Mutex
	closedChannelSubscribers    []chan ClosedChannelEvent[T]
}

func newPriorityChannel[T any](ctx context.Context, compositeChannel selectable.Channel[T], options ...func(*PriorityChannelOptions)) *PriorityChannel[T] {
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	ctx, cancel := context.WithCancel(ctx)
	var l *psync.Lock
	var blockWaitAllChannelsTracker *psync.RepeatingStateTracker
	if pcOptions.isSynchronized == nil || *pcOptions.isSynchronized {
		l = psync.NewLock()
		blockWaitAllChannelsTracker = psync.NewRepeatingStateTracker()
	}
	return &PriorityChannel[T]{
		ctx:                         ctx,
		ctxCancel:                   cancel,
		controlC:                    make(chan controlMessage[T]),
		compositeChannel:            compositeChannel,
		channelReceiveWaitInterval:  pcOptions.channelReceiveWaitInterval,
		lock:                        l,
		blockWaitAllChannelsTracker: blockWaitAllChannelsTracker,
	}
}

func (pc *PriorityChannel[T]) Receive() (msg T, channelName string, ok bool) {
	msg, receiveDetails, status := pc.ReceiveEx()
	return msg, receiveDetails.ChannelName, status
}

func (pc *PriorityChannel[T]) ReceiveEx() (msg T, details ReceiveDetails, ok bool) {
	if pc.lock != nil {
		pc.lock.Lock()
		defer pc.lock.Unlock()
	}
	msg, details, status := pc.receiveSingleMessage(context.Background(), false)
	if status != ReceiveSuccess {
		return getZero[T](), details, false
	}
	return msg, details, true
}

func (pc *PriorityChannel[T]) ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus) {
	msg, receiveDetails, status := pc.ReceiveWithContextEx(ctx)
	return msg, receiveDetails.ChannelName, status
}

func (pc *PriorityChannel[T]) ReceiveWithContextEx(ctx context.Context) (msg T, details ReceiveDetails, status ReceiveStatus) {
	if pc.lock != nil {
		gotLock := pc.lock.TryLockWithContext(ctx)
		if !gotLock {
			return getZero[T](), ReceiveDetails{}, ReceiveContextCanceled
		}
		defer pc.lock.Unlock()
	}
	return pc.receiveSingleMessage(ctx, false)
}

func (pc *PriorityChannel[T]) ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus) {
	msg, receiveDetails, status := pc.ReceiveWithDefaultCaseEx()
	return msg, receiveDetails.ChannelName, status
}

func (pc *PriorityChannel[T]) ReceiveWithDefaultCaseEx() (msg T, details ReceiveDetails, status ReceiveStatus) {
	if pc.lock != nil {
		gotLock := psync.TryLockOrExitOnState(pc.lock, pc.blockWaitAllChannelsTracker)
		if !gotLock {
			return getZero[T](), ReceiveDetails{}, ReceiveDefaultCase
		}
		defer pc.lock.Unlock()
	}
	return pc.receiveSingleMessage(context.Background(), true)
}

func (pc *PriorityChannel[T]) NotifyClose(ch chan ClosedChannelEvent[T]) {
	pc.notifyMtx.Lock()
	defer pc.notifyMtx.Unlock()
	pc.closedChannelSubscribers = append(pc.closedChannelSubscribers, ch)
}

type ClosedChannelEvent[T any] struct {
	ChannelName string
	Details     ReceiveDetails
	RecoverFunc func(<-chan T)
}

func (pc *PriorityChannel[T]) notifyClosedChannelSubscribers(channelName string, pathInTree []selectable.ChannelNode) {
	pc.notifyMtx.Lock()
	defer pc.notifyMtx.Unlock()

	for _, subscriber := range pc.closedChannelSubscribers {
		subscriber <- ClosedChannelEvent[T]{
			ChannelName: channelName,
			Details:     toReceiveDetails(channelName, pathInTree),
			RecoverFunc: func(ch <-chan T) {
				// We need to support both following scenarios:
				// - When no priority_channel.Receive operation is called
				// - When priority_channel.Receive was called and is blocked holding the lock and waiting for a message
				// In both cases we should be able to apply the RecoverClosedChannel operation
				// and proceed without waiting indefinitely.
				select {
				// First case is for normal flow, or to support situation when no Receive operation is called
				case pc.lock.Channel() <- struct{}{}:
					// Lock acquired, now we can enable the closed channel
					pc.compositeChannel.RecoverClosedChannel(ch, pathInTree)
					<-pc.lock.Channel()
				// Second case is also for normal flow, or to support situation when Receive has already been called
				// and is block-waiting while holding the lock
				case pc.controlC <- controlMessage[T]{
					messageType: controlMessageTypeRecoverClosedChannel,
					recoverClosedChannelParams: &recoverClosedChannelParams[T]{
						ch:         ch,
						pathInTree: pathInTree,
					}}:
				}
			},
		}
	}
}

type controlMessageType int

const (
	controlMessageTypeUnknown controlMessageType = iota
	controlMessageTypeRecoverClosedChannel
)

type recoverClosedChannelParams[T any] struct {
	ch         <-chan T
	pathInTree []selectable.ChannelNode
}

type controlMessage[T any] struct {
	messageType                controlMessageType
	recoverClosedChannelParams *recoverClosedChannelParams[T]
}

func (pc *PriorityChannel[T]) Close() {
	pc.ctxCancel()
}

func (pc *PriorityChannel[T]) receiveSingleMessage(ctx context.Context, withDefaultCase bool) (msg T, details ReceiveDetails, status ReceiveStatus) {
	select {
	case <-pc.ctx.Done():
		return getZero[T](), ReceiveDetails{}, ReceivePriorityChannelClosed
	case <-ctx.Done():
		return getZero[T](), ReceiveDetails{}, ReceiveContextCanceled
	default:
	}
	msg, channelName, pathInTree, status := pc.doReceiveSingleMessage(ctx, withDefaultCase)
	for status == ReceiveChannelClosed || status == ReceivePriorityChannelClosed {
		pc.compositeChannel.UpdateOnCaseSelected(pathInTree, false)
		if status == ReceiveChannelClosed {
			pc.notifyClosedChannelSubscribers(channelName, pathInTree)
		}
		prevChannelName := channelName
		msg, channelName, pathInTree, status = pc.doReceiveSingleMessage(ctx, withDefaultCase)
		if channelName == prevChannelName && (status == ReceiveChannelClosed || status == ReceivePriorityChannelClosed) {
			// same channel still returned as closed
			break
		}
	}
	return msg, toReceiveDetails(channelName, pathInTree), status
}

func toReceiveDetails(channelName string, pathInTree []selectable.ChannelNode) ReceiveDetails {
	if len(pathInTree) == 0 {
		return ReceiveDetails{
			ChannelName:  channelName,
			ChannelIndex: 0,
		}
	}
	res := ReceiveDetails{
		ChannelName:  channelName,
		ChannelIndex: pathInTree[0].ChannelIndex,
	}
	if len(pathInTree) > 1 {
		res.PathInTree = make([]ChannelNode, 0, len(pathInTree)-1)
		for i := len(pathInTree) - 2; i >= 0; i-- {
			res.PathInTree = append(res.PathInTree, ChannelNode{
				ChannelName:  pathInTree[i].ChannelName,
				ChannelIndex: pathInTree[i+1].ChannelIndex,
			})
		}
	}
	return res
}

type ReceiveDetails struct {
	ChannelName  string
	ChannelIndex int
	PathInTree   []ChannelNode
}

type ChannelNode struct {
	ChannelName  string
	ChannelIndex int
}

func (pc *PriorityChannel[T]) doReceiveSingleMessage(ctx context.Context, withDefaultCase bool) (msg T, channelName string, PathInTree []selectable.ChannelNode, status ReceiveStatus) {
	nextNumOfChannelsToProcess := 1
	for {
		channelsSelectCases, isLastIteration, closedChannel := pc.compositeChannel.NextSelectCases(nextNumOfChannelsToProcess)
		if closedChannel != nil {
			return getZero[T](), closedChannel.ChannelName, closedChannel.PathInTree, ReceivePriorityChannelClosed
		}
		if len(channelsSelectCases) == 0 {
			if isLastIteration {
				return getZero[T](), "", nil, ReceiveNoOpenChannels
			} else {
				nextNumOfChannelsToProcess++
				continue
			}
		}
		chosen, recv, recvOk, selectStatus := pc.selectCasesOfNextIteration(
			pc.ctx,
			ctx,
			channelsSelectCases,
			isLastIteration,
			false,
			withDefaultCase,
			pc.channelReceiveWaitInterval)
		if selectStatus == ReceiveStatusUnknown {
			if chosen == 2 {
				pc.processControlMessage(recv)
				continue
			}
			if len(channelsSelectCases) <= nextNumOfChannelsToProcess {
				nextNumOfChannelsToProcess++
			} else {
				nextNumOfChannelsToProcess = len(channelsSelectCases) + 1
			}
			continue
		} else if selectStatus != ReceiveSuccess {
			return getZero[T](), "", nil, selectStatus
		}
		channelIndex := chosen - 3
		channelName = channelsSelectCases[channelIndex].ChannelName
		pathInTree := channelsSelectCases[channelIndex].PathInTree
		if !recvOk {
			return getZero[T](), channelName, pathInTree, ReceiveChannelClosed
		}
		// Message received successfully
		msg := recv.Interface().(T)
		pc.compositeChannel.UpdateOnCaseSelected(channelsSelectCases[channelIndex].PathInTree, true)
		return msg, channelName, pathInTree, ReceiveSuccess
	}
}

func (pc *PriorityChannel[T]) processControlMessage(recv reflect.Value) {
	controlMsg, ok := recv.Interface().(controlMessage[T])
	if !ok {
		return
	}
	switch controlMsg.messageType {
	case controlMessageTypeRecoverClosedChannel:
		if controlMsg.recoverClosedChannelParams != nil {
			pc.compositeChannel.RecoverClosedChannel(
				controlMsg.recoverClosedChannelParams.ch,
				controlMsg.recoverClosedChannelParams.pathInTree)
		}
	default:
	}
}

func (pc *PriorityChannel[T]) selectCasesOfNextIteration(
	priorityChannelContext context.Context,
	currRequestContext context.Context,
	channelsSelectCases []selectable.SelectCase[T],
	isLastIteration bool,
	blockWaitAllChannels bool,
	withDefaultCase bool,
	channelReceiveWaitInterval *time.Duration) (chosen int, recv reflect.Value, recvOk bool, status ReceiveStatus) {
	if blockWaitAllChannels && pc.blockWaitAllChannelsTracker != nil {
		pc.blockWaitAllChannelsTracker.Broadcast()
		defer pc.blockWaitAllChannelsTracker.Reset()
	}

	selectCases := make([]reflect.SelectCase, 0, len(channelsSelectCases)+3)
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(priorityChannelContext.Done()),
	})
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(currRequestContext.Done()),
	})
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(pc.controlC),
	})
	for _, sc := range channelsSelectCases {
		selectCases = append(selectCases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(sc.MsgsC),
		})
	}
	if !isLastIteration || withDefaultCase || !blockWaitAllChannels {
		selectCases = append(selectCases, getDefaultSelectCaseWithWaitInterval(channelReceiveWaitInterval))
	}

	chosen, recv, recvOk = reflect.Select(selectCases)
	switch chosen {
	case 0:
		// context of the priority channel is done
		return chosen, recv, recvOk, ReceivePriorityChannelClosed
	case 1:
		// context of the specific request is done
		return chosen, recv, recvOk, ReceiveContextCanceled
	case 2:
		// control message has arrived
		return chosen, recv, recvOk, ReceiveStatusUnknown

	case len(selectCases) - 1:
		if !isLastIteration {
			// Default case - go to next iteration to increase the range of allowed minimal priority channels
			// on last iteration - blocking wait on all receive channels without default case
			return chosen, recv, recvOk, ReceiveStatusUnknown
		} else if !blockWaitAllChannels {
			// last call is block wait on all channels
			blockWaitAllChannels = true
			return pc.selectCasesOfNextIteration(
				priorityChannelContext, currRequestContext, channelsSelectCases, isLastIteration,
				blockWaitAllChannels, withDefaultCase, channelReceiveWaitInterval)
		} else if withDefaultCase {
			return chosen, recv, recvOk, ReceiveDefaultCase
		}
	}
	return chosen, recv, recvOk, ReceiveSuccess
}

func getDefaultSelectCaseWithWaitInterval(channelReceiveWaitInterval *time.Duration) reflect.SelectCase {
	waitInterval := defaultChannelReceiveWaitInterval
	if channelReceiveWaitInterval != nil {
		waitInterval = *channelReceiveWaitInterval
	}
	if waitInterval > 0 {
		return reflect.SelectCase{
			// The default behavior without a wait interval may not work
			// if receiving a message from the channel takes some time.
			// In such cases, a short wait is needed to ensure that the default case
			// is not triggered while messages are still available in the channel.
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(time.After(waitInterval)),
		}
	}
	return reflect.SelectCase{Dir: reflect.SelectDefault}
}
