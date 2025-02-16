package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/channels"
)

func WrapAsPriorityChannel[T any](ctx context.Context, channelName string, msgsC <-chan T) (PriorityChannel[T], error) {
	if channelName == "" {
		return nil, ErrEmptyChannelName
	}
	return &wrappedChannel[T]{ctx: ctx, channelName: channelName, msgsC: msgsC}, nil
}

type wrappedChannel[T any] struct {
	ctx         context.Context
	channelName string
	msgsC       <-chan T
}

func (w *wrappedChannel[T]) Receive() (msg T, channelName string, ok bool) {
	select {
	case <-w.ctx.Done():
		return getZero[T](), "", false
	case msg, ok = <-w.msgsC:
		return msg, w.channelName, ok
	}
}

func (w *wrappedChannel[T]) ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus) {
	select {
	case <-w.ctx.Done():
		return getZero[T](), "", ReceivePriorityChannelCancelled
	case <-ctx.Done():
		return getZero[T](), "", ReceiveContextCancelled
	case msg, ok := <-w.msgsC:
		if !ok {
			return getZero[T](), w.channelName, ReceiveChannelClosed
		}
		return msg, w.channelName, ReceiveSuccess
	}
}

func (w *wrappedChannel[T]) ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus) {
	select {
	case <-w.ctx.Done():
		return getZero[T](), "", ReceivePriorityChannelCancelled
	case msg, ok := <-w.msgsC:
		if !ok {
			return getZero[T](), w.channelName, ReceiveChannelClosed
		}
		return msg, w.channelName, ReceiveSuccess
	default:
		return getZero[T](), "", ReceiveDefaultCase
	}
}

func (w *wrappedChannel[T]) AsSelectableChannelWithPriority(name string, priority int) channels.ChannelWithPriority[T] {
	return &wrapCompositeChannelWithPriorityWithContext[T]{
		ctx:                 w.ctx,
		ChannelWithPriority: channels.NewChannelWithPriority(name, w.msgsC, priority),
	}
}

func (w *wrappedChannel[T]) AsSelectableChannelWithFreqRatio(name string, freqRatio int) channels.ChannelWithFreqRatio[T] {
	return &wrapCompositeChannelWithFreqRatioWithContext[T]{
		ctx:                  w.ctx,
		ChannelWithFreqRatio: channels.NewChannelWithFreqRatio(name, w.msgsC, freqRatio),
	}
}

type wrapCompositeChannelWithPriorityWithContext[T any] struct {
	channels.ChannelWithPriority[T]
	ctx context.Context
}

func (w *wrapCompositeChannelWithPriorityWithContext[T]) NextSelectCases(upto int) (selectCases []channels.SelectCase[T], isLastIteration bool, closedChannel *channels.ClosedChannelDetails) {
	select {
	case <-w.ctx.Done():
		return nil, true, &channels.ClosedChannelDetails{
			ChannelName: w.ChannelName(),
			PathInTree:  nil,
		}
	default:
		return w.ChannelWithPriority.NextSelectCases(upto)
	}
}

type wrapCompositeChannelWithFreqRatioWithContext[T any] struct {
	channels.ChannelWithFreqRatio[T]
	ctx context.Context
}

func (w *wrapCompositeChannelWithFreqRatioWithContext[T]) NextSelectCases(upto int) (selectCases []channels.SelectCase[T], isLastIteration bool, closedChannel *channels.ClosedChannelDetails) {
	select {
	case <-w.ctx.Done():
		return nil, true, &channels.ClosedChannelDetails{
			ChannelName: w.ChannelName(),
			PathInTree:  nil,
		}
	default:
		return w.ChannelWithFreqRatio.NextSelectCases(upto)
	}
}
