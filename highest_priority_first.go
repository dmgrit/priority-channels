package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
)

func NewByHighestAlwaysFirst[T any](ctx context.Context,
	channelsWithPriorities []channels.ChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	selectableChannels := make([]selectable.ChannelWithWeight[T, int], 0, len(channelsWithPriorities))
	for _, c := range channelsWithPriorities {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithWeight(
			channels.NewChannelWithWeight[T, int](c.ChannelName(), c.MsgsC(), c.Priority()),
		))
	}
	strategy := priority_strategies.NewByHighestAlwaysFirst()
	return newByStrategy(ctx, strategy, selectableChannels, options...)
}
