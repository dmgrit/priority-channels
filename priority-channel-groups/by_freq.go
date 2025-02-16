package priority_channel_groups

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func CombineByFrequencyRatio[T any](ctx context.Context,
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T],
	options ...func(*priority_channels.PriorityChannelOptions)) (priority_channels.PriorityChannel[T], error) {
	channels := newPriorityChannelsGroupByFreqRatio[T](priorityChannelsWithFreqRatio)
	return priority_channels.NewByFrequencyRatio[T](ctx, channels, options...)
}

type PriorityChannelWithFreqRatio[T any] interface {
	Name() string
	PriorityChannel() priority_channels.PriorityChannel[T]
	FreqRatio() int
}

type priorityChannelWithFreqRatio[T any] struct {
	name            string
	priorityChannel priority_channels.PriorityChannel[T]
	freqRatio       int
}

func (c *priorityChannelWithFreqRatio[T]) Name() string {
	return c.name
}

func (c *priorityChannelWithFreqRatio[T]) PriorityChannel() priority_channels.PriorityChannel[T] {
	return c.priorityChannel
}

func (c *priorityChannelWithFreqRatio[T]) FreqRatio() int {
	return c.freqRatio
}

func NewPriorityChannelWithFreqRatio[T any](name string, priorityChannel priority_channels.PriorityChannel[T], freqRatio int) PriorityChannelWithFreqRatio[T] {
	return &priorityChannelWithFreqRatio[T]{
		name:            name,
		priorityChannel: priorityChannel,
		freqRatio:       freqRatio,
	}
}

func newPriorityChannelsGroupByFreqRatio[T any](
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T]) []channels.ChannelWithFreqRatio[T] {
	res := make([]channels.ChannelWithFreqRatio[T], 0, len(priorityChannelsWithFreqRatio))

	for _, q := range priorityChannelsWithFreqRatio {
		res = append(res, q.PriorityChannel().AsSelectableChannelWithFreqRatio(q.Name(), q.FreqRatio()))
	}
	sort.Slice(res, func(i int, j int) bool {
		return res[i].FreqRatio() > res[j].FreqRatio()
	})
	return res
}
