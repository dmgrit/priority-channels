package priority_channels

import (
	"context"
	"reflect"
	"time"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

type PriorityChannel[T any] struct {
	ctx                         context.Context
	ctxCancel                   context.CancelFunc
	compositeChannel            selectable.Channel[T]
	channelReceiveWaitInterval  *time.Duration
	lock                        *lock
	blockWaitAllChannelsTracker *repeatingStateTracker
}

func newPriorityChannel[T any](ctx context.Context, compositeChannel selectable.Channel[T], options ...func(*PriorityChannelOptions)) *PriorityChannel[T] {
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	ctx, cancel := context.WithCancel(ctx)
	var l *lock
	var blockWaitAllChannelsTracker *repeatingStateTracker
	if pcOptions.isSynchronized != nil && *pcOptions.isSynchronized {
		l = newLock()
		blockWaitAllChannelsTracker = newRepeatingStateTracker()
	}
	return &PriorityChannel[T]{
		ctx:                         ctx,
		ctxCancel:                   cancel,
		compositeChannel:            compositeChannel,
		channelReceiveWaitInterval:  pcOptions.channelReceiveWaitInterval,
		lock:                        l,
		blockWaitAllChannelsTracker: blockWaitAllChannelsTracker,
	}
}

func (pc *PriorityChannel[T]) Receive() (msg T, channelName string, ok bool) {
	if pc.lock != nil {
		pc.lock.Lock()
		defer pc.lock.Unlock()
	}
	msg, channelName, status := pc.receiveSingleMessage(context.Background(), false)
	if status != ReceiveSuccess {
		return getZero[T](), channelName, false
	}
	return msg, channelName, true
}

func (pc *PriorityChannel[T]) ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus) {
	if pc.lock != nil {
		gotLock := pc.lock.TryLockWithContext(ctx)
		if !gotLock {
			return getZero[T](), "", ReceiveContextCancelled
		}
		defer pc.lock.Unlock()
	}
	return pc.receiveSingleMessage(ctx, false)
}

func (pc *PriorityChannel[T]) ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus) {
	if pc.lock != nil {
		gotLock := pc.waitForLockOrExitOnBlockWaitAllChannels()
		if !gotLock {
			return getZero[T](), "", ReceiveDefaultCase
		}
		defer pc.lock.Unlock()
	}
	return pc.receiveSingleMessage(context.Background(), true)
}

func (pc *PriorityChannel[T]) waitForLockOrExitOnBlockWaitAllChannels() bool {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if pc.blockWaitAllChannelsTracker.Await(ctx) {
			cancel()
		}
	}()
	if !pc.lock.TryLockWithContext(ctx) {
		return false
	}
	cancel()
	return true
}

func (pc *PriorityChannel[T]) Close() {
	pc.ctxCancel()
}

func (pc *PriorityChannel[T]) receiveSingleMessage(ctx context.Context, withDefaultCase bool) (msg T, channelName string, status ReceiveStatus) {
	select {
	case <-pc.ctx.Done():
		return getZero[T](), "", ReceivePriorityChannelClosed
	case <-ctx.Done():
		return getZero[T](), "", ReceiveContextCancelled
	default:
	}
	msg, channelName, pathInTree, status := pc.doReceiveSingleMessage(ctx, withDefaultCase)
	for status == ReceiveChannelClosed || status == ReceivePriorityChannelClosed {
		pc.compositeChannel.UpdateOnCaseSelected(pathInTree, false)
		prevChannelName := channelName
		msg, channelName, pathInTree, status = pc.doReceiveSingleMessage(ctx, withDefaultCase)
		if channelName == prevChannelName && (status == ReceiveChannelClosed || status == ReceivePriorityChannelClosed) {
			// same channel still returned as closed
			break
		}
	}
	return msg, channelName, status
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
		} else if len(channelsSelectCases) <= nextNumOfChannelsToProcess {
			nextNumOfChannelsToProcess++
		} else {
			nextNumOfChannelsToProcess = len(channelsSelectCases) + 1
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
			continue
		} else if selectStatus != ReceiveSuccess {
			return getZero[T](), "", nil, selectStatus
		}
		channelIndex := chosen - 2
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
		return chosen, recv, recvOk, ReceiveContextCancelled

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
