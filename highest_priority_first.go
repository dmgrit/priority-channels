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
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	selectableChannels := make([]selectable.ChannelWithWeight[T, int], 0, len(channelsWithPriorities))
	for _, c := range channelsWithPriorities {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithWeight(
			channels.NewChannelWithWeight[T, int](c.ChannelName(), c.MsgsC(), c.Priority()),
		))
	}
	_, err := getFrequencyStrategy(LevelNew, pcOptions.frequencyMode, pcOptions.frequencyMethod, len(selectableChannels))
	if err != nil {
		return nil, err
	}
	strategy := priority_strategies.NewByHighestAlwaysFirst(priority_strategies.WithFrequencyStrategyGenerator(func(numChannels int) priority_strategies.FrequencyStrategy {
		frequencyStrategy, _ := getFrequencyStrategy(LevelNew, pcOptions.frequencyMode, pcOptions.frequencyMethod, numChannels)
		return frequencyStrategy
	}))
	return newByStrategy(ctx, strategy, selectableChannels, options...)
}
