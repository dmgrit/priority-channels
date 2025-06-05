package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies"
)

func CombineByStrategy[T any, W any](ctx context.Context,
	strategy strategies.PrioritizationStrategy[W],
	priorityChannelsWithWeight []PriorityChannelWithWeight[T, W],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	channels := toSelectableChannelsWithWeight[T](priorityChannelsWithWeight)
	return newByStrategy(ctx, strategy, channels, options...)
}

type PriorityChannelWithWeight[T any, W any] struct {
	name            string
	priorityChannel *PriorityChannel[T]
	weight          W
}

func (c *PriorityChannelWithWeight[T, W]) Name() string {
	return c.name
}

func (c *PriorityChannelWithWeight[T, W]) PriorityChannel() *PriorityChannel[T] {
	return c.priorityChannel
}

func (c *PriorityChannelWithWeight[T, W]) Weight() W {
	return c.weight
}

func NewPriorityChannelWithWeight[T any, W any](name string, priorityChannel *PriorityChannel[T], weight W) PriorityChannelWithWeight[T, W] {
	return PriorityChannelWithWeight[T, W]{
		name:            name,
		priorityChannel: priorityChannel,
		weight:          weight,
	}
}

func toSelectableChannelsWithWeight[T any, W any](
	priorityChannelsWithWeight []PriorityChannelWithWeight[T, W]) []selectable.ChannelWithWeight[T, W] {
	res := make([]selectable.ChannelWithWeight[T, W], 0, len(priorityChannelsWithWeight))
	for _, q := range priorityChannelsWithWeight {
		priorityChannel := q.PriorityChannel().clone()
		res = append(res, asSelectableChannelWithWeight(priorityChannel, q.Name(), q.Weight()))
	}
	return res
}
