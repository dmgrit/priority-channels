package priority_channels

import (
	"context"
	"reflect"
	"sync/atomic"
	"time"
)

type PriorityChannel[T any] interface {
	Receive() (msg T, channelName string, ok bool)
	ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus)
	ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus)
}

type PriorityChannelWithContext[T any] interface {
	PriorityChannel[T]
	Context() context.Context
}

type ReceiveStatus int

const (
	ReceiveSuccess ReceiveStatus = iota
	ReceiveContextCancelled
	ReceiveChannelClosed
	ReceiveDefaultCase
	ReceivePriorityChannelCancelled
	ReceiveStatusUnknown
)

func (r ReceiveStatus) ExitReason() ExitReason {
	switch r {
	case ReceiveChannelClosed:
		return ChannelClosed
	case ReceivePriorityChannelCancelled:
		return PriorityChannelCancelled
	default:
		return UnknownExitReason
	}
}

type ExitReason int

const (
	ChannelClosed ExitReason = iota
	PriorityChannelCancelled
	UnknownExitReason
)

type PriorityChannelOptions struct {
	channelReceiveWaitInterval *time.Duration
}

const defaultChannelReceiveWaitInterval = 100 * time.Microsecond

func ChannelWaitInterval(d time.Duration) func(opt *PriorityChannelOptions) {
	return func(opt *PriorityChannelOptions) {
		opt.channelReceiveWaitInterval = &d
	}
}

type ChannelWithUnderlyingClosedChannelDetails interface {
	// An empty channel name indicates that the current channel is closed.
	// A non-empty channel name indicates that some underlying descendant channel is closed.
	GetUnderlyingClosedChannelDetails() (channelName string, closeStatus ReceiveStatus)
}

type ReadinessChecker interface {
	IsReady() bool
}

func ProcessPriorityChannelMessages[T any](
	msgReceiver PriorityChannel[T],
	msgProcessor func(ctx context.Context, msg T, channelName string)) ExitReason {
	for {
		// There is no context per-message, but there is a single context for the entire priority-channel
		// On receiving the message we do not pass any specific context,
		// but on processing the message we pass the priority-channel context
		msg, channelName, status := msgReceiver.ReceiveWithContext(context.Background())
		if status != ReceiveSuccess {
			return status.ExitReason()
		}
		ctx := context.Background()
		msgReceiverWithContext, ok := msgReceiver.(PriorityChannelWithContext[T])
		if ok {
			ctx = msgReceiverWithContext.Context()
		}
		msgProcessor(ctx, msg, channelName)
	}
}

func getZero[T any]() T {
	var result T
	return result
}

func selectCasesOfNextIteration(
	priorityChannelContext context.Context,
	currRequestContext context.Context,
	fnPrepareChannelsSelectCases func(currIterationIndex int) []reflect.SelectCase,
	currIterationIndex int,
	lastIterationIndex int,
	withDefaultCase bool,
	isPreparing *atomic.Bool,
	channelReceiveWaitInterval *time.Duration) (chosen int, recv reflect.Value, recvOk bool, status ReceiveStatus) {

	isLastIteration := currIterationIndex == lastIterationIndex
	channelsSelectCases := fnPrepareChannelsSelectCases(currIterationIndex)

	selectCases := make([]reflect.SelectCase, 0, len(channelsSelectCases)+3)
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(priorityChannelContext.Done()),
	})
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(currRequestContext.Done()),
	})
	selectCases = append(selectCases, channelsSelectCases...)
	if !isLastIteration || withDefaultCase || isPreparing.Load() {
		selectCases = append(selectCases, getDefaultSelectCaseWithWaitInterval(channelReceiveWaitInterval))
	}

	chosen, recv, recvOk = reflect.Select(selectCases)
	switch chosen {
	case 0:
		// context of the priority channel is done
		return chosen, recv, recvOk, ReceivePriorityChannelCancelled
	case 1:
		// context of the specific request is done
		return chosen, recv, recvOk, ReceiveContextCancelled

	case len(selectCases) - 1:
		if !isLastIteration {
			// Default case - go to next iteration to increase the range of allowed minimal priority channels
			// on last iteration - blocking wait on all receive channels without default case
			return chosen, recv, recvOk, ReceiveStatusUnknown
		} else if withDefaultCase {
			return chosen, recv, recvOk, ReceiveDefaultCase
		} else if isPreparing.Load() {
			isPreparing.Store(false)
			// recursive call for last iteration - this time will issue a blocking wait on all channels
			return selectCasesOfNextIteration(priorityChannelContext, currRequestContext,
				fnPrepareChannelsSelectCases, currIterationIndex, lastIterationIndex,
				withDefaultCase, isPreparing, channelReceiveWaitInterval)
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

func waitForReadyStatus(ch interface{}) {
	if ch == nil {
		return
	}
	if checker, ok := ch.(ReadinessChecker); ok {
		for !checker.IsReady() {
			time.Sleep(100 * time.Microsecond)
		}
	}
	return
}
