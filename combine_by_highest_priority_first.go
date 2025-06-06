package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
)

func CombineByHighestAlwaysFirst[T any](ctx context.Context,
	priorityChannelsWithPriority []PriorityChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	channels, channelsWeights := toSelectableChannelsWithWeightByPriority[T](priorityChannelsWithPriority)
	_, err := getFrequencyStrategy(levelCombine, pcOptions.frequencyMode, pcOptions.frequencyMethod, len(channels))
	if err != nil {
		return nil, err
	}
	strategy := priority_strategies.NewByHighestAlwaysFirst(priority_strategies.WithFrequencyStrategyGenerator(func(numChannels int) priority_strategies.FrequencyStrategy {
		frequencyStrategy, _ := getFrequencyStrategy(levelCombine, pcOptions.frequencyMode, pcOptions.frequencyMethod, numChannels)
		return frequencyStrategy
	}))
	return newByStrategy(ctx, strategy, channels, channelsWeights, options...)
}

type PriorityChannelWithPriority[T any] struct {
	name            string
	priorityChannel *PriorityChannel[T]
	priority        int
}

func (c *PriorityChannelWithPriority[T]) Name() string {
	return c.name
}

func (c *PriorityChannelWithPriority[T]) PriorityChannel() *PriorityChannel[T] {
	return c.priorityChannel
}

func (c *PriorityChannelWithPriority[T]) Priority() int {
	return c.priority
}

func NewPriorityChannelWithPriority[T any](name string, priorityChannel *PriorityChannel[T], priority int) PriorityChannelWithPriority[T] {
	return PriorityChannelWithPriority[T]{
		name:            name,
		priorityChannel: priorityChannel,
		priority:        priority,
	}
}

func toSelectableChannelsWithWeightByPriority[T any](
	priorityChannelsWithPriority []PriorityChannelWithPriority[T]) ([]selectable.Channel[T], []int) {
	res := make([]selectable.Channel[T], 0, len(priorityChannelsWithPriority))
	resWeights := make([]int, 0, len(priorityChannelsWithPriority))
	for _, c := range priorityChannelsWithPriority {
		priorityChannel := c.PriorityChannel().clone()
		res = append(res, asSelectableChannelWithName(priorityChannel, c.Name()))
		resWeights = append(resWeights, c.Priority())
	}
	return res, resWeights
}
