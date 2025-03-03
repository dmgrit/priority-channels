package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func WrapAsPriorityChannel[T any](ctx context.Context, channelName string, msgsC <-chan T, options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	if channelName == "" {
		return nil, ErrEmptyChannelName
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	compositeChannel := &wrappedChannel[T]{
		ctx:                        ctx,
		channelName:                channelName,
		msgsC:                      msgsC,
		autoDisableOnClosedChannel: pcOptions.autoDisableClosedChannels,
	}
	return newPriorityChannel(ctx, compositeChannel, options...), nil
}

type wrappedChannel[T any] struct {
	ctx                        context.Context
	channelName                string
	msgsC                      <-chan T
	autoDisableOnClosedChannel bool
	disabled                   bool
}

func (w *wrappedChannel[T]) ChannelName() string {
	return w.channelName
}

func (w *wrappedChannel[T]) NextSelectCases(upto int) (selectCases []selectable.SelectCase[T], isLastIteration bool, closedChannel *selectable.ClosedChannelDetails) {
	select {
	case <-w.ctx.Done():
		return nil, true, &selectable.ClosedChannelDetails{
			ChannelName: "",
			PathInTree:  nil,
		}
	default:
		if w.disabled {
			return nil, true, nil
		}
		return []selectable.SelectCase[T]{
			{
				ChannelName: w.channelName,
				MsgsC:       w.msgsC,
			},
		}, true, nil
	}
}

func (c *wrappedChannel[T]) UpdateOnCaseSelected(pathInTree []selectable.ChannelNode, recvOK bool) {
	if !recvOK && c.autoDisableOnClosedChannel {
		c.disabled = true
	}
}

func (c *wrappedChannel[T]) Validate() error {
	if c.channelName == "" {
		return ErrEmptyChannelName
	}
	return nil
}
