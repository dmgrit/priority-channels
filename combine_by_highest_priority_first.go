package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies"
)

func CombineByHighestPriorityFirst[T any](ctx context.Context,
	priorityChannelsWithPriority []PriorityChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	channels := toSelectableChannelsWithWeightByPriority[T](priorityChannelsWithPriority)
	strategy := strategies.NewByHighestAlwaysFirst()
	return newByStrategy(ctx, strategy, channels, options...)
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
	priorityChannelsWithPriority []PriorityChannelWithPriority[T]) []selectable.ChannelWithWeight[T, int] {
	res := make([]selectable.ChannelWithWeight[T, int], 0, len(priorityChannelsWithPriority))
	for _, q := range priorityChannelsWithPriority {
		priorityChannel := q.PriorityChannel()
		res = append(res, asSelectableChannelWithWeight(priorityChannel, q.Name(), q.Priority()))
	}
	return res
}
