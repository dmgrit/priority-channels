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
		ctx:         ctx,
		channelName: channelName,
		msgsC:       msgsC,
	}
	return &PriorityChannel[T]{
		ctx:                        ctx,
		compositeChannel:           compositeChannel,
		channelReceiveWaitInterval: pcOptions.channelReceiveWaitInterval,
	}, nil
}

type wrappedChannel[T any] struct {
	ctx         context.Context
	channelName string
	msgsC       <-chan T
}

func (w *wrappedChannel[T]) ChannelName() string {
	return w.channelName
}

func (w *wrappedChannel[T]) NextSelectCases(upto int) (selectCases []selectable.SelectCase[T], isLastIteration bool, closedChannel *selectable.ClosedChannelDetails) {
	select {
	case <-w.ctx.Done():
		return nil, true, &selectable.ClosedChannelDetails{
			ChannelName: w.channelName,
			PathInTree:  nil,
		}
	default:
		return []selectable.SelectCase[T]{
			{
				ChannelName: w.channelName,
				MsgsC:       w.msgsC,
			},
		}, true, nil
	}
}

func (c *wrappedChannel[T]) UpdateOnCaseSelected(pathInTree []selectable.ChannelNode) {}
