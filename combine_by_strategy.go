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
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	channels, channelsWeights := toSelectableChannelsWithWeight[T](priorityChannelsWithWeight, pcOptions.combineWithoutClone)
	return newByStrategy(ctx, strategy, channels, channelsWeights, options...)
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
	priorityChannelsWithWeight []PriorityChannelWithWeight[T, W], noClone bool) ([]selectable.Channel[T], []W) {
	res := make([]selectable.Channel[T], 0, len(priorityChannelsWithWeight))
	resWeights := make([]W, 0, len(priorityChannelsWithWeight))
	for _, c := range priorityChannelsWithWeight {
		var priorityChannel *PriorityChannel[T]
		if noClone {
			priorityChannel = c.PriorityChannel()
		} else {
			priorityChannel = c.PriorityChannel().clone()
		}
		res = append(res, asSelectableChannelWithName(priorityChannel, c.Name()))
		resWeights = append(resWeights, c.Weight())
	}
	return res, resWeights
}
