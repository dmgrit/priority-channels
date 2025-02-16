package priority_channel_groups

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func CombineByHighestPriorityFirst[T any](ctx context.Context,
	priorityChannelsWithPriority []PriorityChannelWithPriority[T],
	options ...func(*priority_channels.PriorityChannelOptions)) (priority_channels.PriorityChannel[T], error) {
	channels := newPriorityChannelsGroupByHighestPriorityFirst[T](priorityChannelsWithPriority)
	return priority_channels.NewByHighestAlwaysFirst[T](ctx, channels, options...)
}

type PriorityChannelWithPriority[T any] interface {
	Name() string
	PriorityChannel() priority_channels.PriorityChannel[T]
	Priority() int
}

type priorityChannelWithPriority[T any] struct {
	name            string
	priorityChannel priority_channels.PriorityChannel[T]
	priority        int
}

func (c *priorityChannelWithPriority[T]) Name() string {
	return c.name
}

func (c *priorityChannelWithPriority[T]) PriorityChannel() priority_channels.PriorityChannel[T] {
	return c.priorityChannel
}

func (c *priorityChannelWithPriority[T]) Priority() int {
	return c.priority
}

func NewPriorityChannelWithPriority[T any](name string, priorityChannel priority_channels.PriorityChannel[T], priority int) PriorityChannelWithPriority[T] {
	return &priorityChannelWithPriority[T]{
		name:            name,
		priorityChannel: priorityChannel,
		priority:        priority,
	}
}

func newPriorityChannelsGroupByHighestPriorityFirst[T any](
	priorityChannelsWithPriority []PriorityChannelWithPriority[T]) []channels.ChannelWithPriority[T] {
	res := make([]channels.ChannelWithPriority[T], 0, len(priorityChannelsWithPriority))

	for _, q := range priorityChannelsWithPriority {
		res = append(res, q.PriorityChannel().AsSelectableChannelWithPriority(q.Name(), q.Priority()))
	}
	sort.Slice(res, func(i int, j int) bool {
		return res[i].Priority() > res[j].Priority()
	})
	return res
}
