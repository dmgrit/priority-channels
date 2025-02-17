package priority_channels

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func CombineByHighestPriorityFirst[T any](ctx context.Context,
	priorityChannelsWithPriority []PriorityChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	channels := newPriorityChannelsGroupByHighestPriorityFirst[T](priorityChannelsWithPriority)
	return newByHighestAlwaysFirst[T](ctx, channels, options...)
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

func newPriorityChannelsGroupByHighestPriorityFirst[T any](
	priorityChannelsWithPriority []PriorityChannelWithPriority[T]) []selectable.ChannelWithPriority[T] {
	res := make([]selectable.ChannelWithPriority[T], 0, len(priorityChannelsWithPriority))

	for _, q := range priorityChannelsWithPriority {
		priorityChannel := q.PriorityChannel()
		res = append(res, priorityChannel.asSelectableChannelWithPriority(q.Name(), q.Priority()))
	}
	sort.Slice(res, func(i int, j int) bool {
		return res[i].Priority() > res[j].Priority()
	})
	return res
}
