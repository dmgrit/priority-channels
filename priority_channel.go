package priority_channels

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/dmgrit/priority-channels/channels"
)

type priorityChannel[T any] struct {
	ctx                        context.Context
	compositeChannel           channels.SelectableChannel[T]
	channelReceiveWaitInterval *time.Duration
}

func (pc *priorityChannel[T]) Receive() (msg T, channelName string, ok bool) {
	msg, channelName, status := pc.receiveSingleMessage(context.Background(), false)
	if status != ReceiveSuccess {
		return getZero[T](), channelName, false
	}
	return msg, channelName, true
}

func (pc *priorityChannel[T]) ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus) {
	return pc.receiveSingleMessage(ctx, false)
}

func (pc *priorityChannel[T]) ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus) {
	return pc.receiveSingleMessage(context.Background(), true)
}

func (pc *priorityChannel[T]) Context() context.Context {
	return pc.ctx
}

func (pc *priorityChannel[T]) AsSelectableChannelWithPriority(name string, priority int) channels.ChannelWithPriority[T] {
	return &wrapCompositeChannelWithNameAndPriority[T]{
		overrideCompositeChannelName: overrideCompositeChannelName[T]{
			ctx:     pc.ctx,
			name:    name,
			channel: pc.compositeChannel,
		},
		priority: priority,
	}
}

func (pc *priorityChannel[T]) AsSelectableChannelWithFreqRatio(name string, freqRatio int) channels.ChannelWithFreqRatio[T] {
	return &wrapCompositeChannelWithNameAndFreqRatio[T]{
		overrideCompositeChannelName: overrideCompositeChannelName[T]{
			ctx:     pc.ctx,
			name:    name,
			channel: pc.compositeChannel,
		},
		freqRatio: freqRatio,
	}
}

type wrapCompositeChannelWithNameAndPriority[T any] struct {
	overrideCompositeChannelName[T]
	priority int
}

func (w *wrapCompositeChannelWithNameAndPriority[T]) Priority() int {
	return w.priority
}

func (w *wrapCompositeChannelWithNameAndPriority[T]) Validate() error {
	if w.priority < 0 {
		return errors.New("priority cannot be negative")
	}
	return w.overrideCompositeChannelName.Validate()
}

type wrapCompositeChannelWithNameAndFreqRatio[T any] struct {
	overrideCompositeChannelName[T]
	freqRatio int
}

func (w *wrapCompositeChannelWithNameAndFreqRatio[T]) FreqRatio() int {
	return w.freqRatio
}

func (w *wrapCompositeChannelWithNameAndFreqRatio[T]) Validate() error {
	if w.freqRatio <= 0 {
		return errors.New("frequency ratio must be greater than 0")
	}
	return w.overrideCompositeChannelName.Validate()
}

type overrideCompositeChannelName[T any] struct {
	ctx     context.Context
	name    string
	channel channels.SelectableChannel[T]
}

func (oc *overrideCompositeChannelName[T]) ChannelName() string {
	return oc.name
}

func (oc *overrideCompositeChannelName[T]) NextSelectCases(upto int) ([]channels.SelectCase[T], bool, *channels.ClosedChannelDetails) {
	select {
	case <-oc.ctx.Done():
		return nil, true, &channels.ClosedChannelDetails{
			ChannelName: oc.ChannelName(),
			PathInTree:  nil,
		}
	default:
		res, allSelected, closedChannel := oc.channel.NextSelectCases(upto)
		if closedChannel != nil {
			closedChannel.PathInTree[len(closedChannel.PathInTree)-1].ChannelName = oc.name
			return res, allSelected, closedChannel
		}
		for i, sc := range res {
			if len(sc.PathInTree) > 0 {
				res[i].PathInTree[len(res[i].PathInTree)-1].ChannelName = oc.name
			}
		}
		return res, allSelected, nil
	}
}

func (oc *overrideCompositeChannelName[T]) UpdateOnCaseSelected(pathInTree []channels.ChannelNode) {
	oc.channel.UpdateOnCaseSelected(pathInTree)
}

func (oc *overrideCompositeChannelName[T]) Validate() error {
	if oc.name == "" {
		return ErrEmptyChannelName
	}
	return oc.channel.Validate()
}

func (pc *priorityChannel[T]) receiveSingleMessage(ctx context.Context, withDefaultCase bool) (msg T, channelName string, status ReceiveStatus) {
	currNumOfChannelsToProcess := 0
	for {
		currNumOfChannelsToProcess++
		channelsSelectCases, isLastIteration, closedChannel := pc.compositeChannel.NextSelectCases(currNumOfChannelsToProcess)
		if closedChannel != nil {
			return getZero[T](), closedChannel.ChannelName, ReceivePriorityChannelCancelled
		}
		chosen, recv, recvOk, selectStatus := pc.selectCasesOfNextIteration(
			pc.ctx,
			ctx,
			channelsSelectCases,
			isLastIteration,
			withDefaultCase,
			pc.channelReceiveWaitInterval)
		if selectStatus == ReceiveStatusUnknown {
			continue
		} else if selectStatus != ReceiveSuccess {
			return getZero[T](), "", selectStatus
		}
		channelIndex := chosen - 2
		channelName = channelsSelectCases[channelIndex].ChannelName
		if !recvOk {
			return getZero[T](), channelName, ReceiveChannelClosed
		}
		// Message received successfully
		msg := recv.Interface().(T)
		pc.compositeChannel.UpdateOnCaseSelected(channelsSelectCases[channelIndex].PathInTree)
		return msg, channelName, ReceiveSuccess
	}
}

func (pc *priorityChannel[T]) selectCasesOfNextIteration(
	priorityChannelContext context.Context,
	currRequestContext context.Context,
	channelsSelectCases []channels.SelectCase[T],
	isLastIteration bool,
	withDefaultCase bool,
	channelReceiveWaitInterval *time.Duration) (chosen int, recv reflect.Value, recvOk bool, status ReceiveStatus) {

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
	if !isLastIteration || withDefaultCase {
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
