package priority_channels

import "context"

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
