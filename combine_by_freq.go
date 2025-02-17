package priority_channels

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func CombineByFrequencyRatio[T any](ctx context.Context,
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	channels := newPriorityChannelsGroupByFreqRatio[T](priorityChannelsWithFreqRatio)
	return newByFrequencyRatio[T](ctx, channels, options...)
}

type PriorityChannelWithFreqRatio[T any] struct {
	name            string
	priorityChannel *PriorityChannel[T]
	freqRatio       int
}

func (c *PriorityChannelWithFreqRatio[T]) Name() string {
	return c.name
}

func (c *PriorityChannelWithFreqRatio[T]) PriorityChannel() *PriorityChannel[T] {
	return c.priorityChannel
}

func (c *PriorityChannelWithFreqRatio[T]) FreqRatio() int {
	return c.freqRatio
}

func NewPriorityChannelWithFreqRatio[T any](name string, priorityChannel *PriorityChannel[T], freqRatio int) PriorityChannelWithFreqRatio[T] {
	return PriorityChannelWithFreqRatio[T]{
		name:            name,
		priorityChannel: priorityChannel,
		freqRatio:       freqRatio,
	}
}

func newPriorityChannelsGroupByFreqRatio[T any](
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T]) []selectable.ChannelWithFreqRatio[T] {
	res := make([]selectable.ChannelWithFreqRatio[T], 0, len(priorityChannelsWithFreqRatio))

	for _, q := range priorityChannelsWithFreqRatio {
		priorityChannel := q.PriorityChannel()
		res = append(res, priorityChannel.asSelectableChannelWithFreqRatio(q.Name(), q.FreqRatio()))
	}
	sort.Slice(res, func(i int, j int) bool {
		return res[i].FreqRatio() > res[j].FreqRatio()
	})
	return res
}
