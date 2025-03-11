package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func CombineByFrequencyRatio[T any](ctx context.Context,
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	strategy, probabilityStrategy, err := getFrequencyStrategy(options...)
	if err != nil {
		return nil, err
	}
	if probabilityStrategy != nil {
		probabilityChannels := toProbabilitySelectableChannelsWithWeightByFreqRatio(priorityChannelsWithFreqRatio)
		return newByStrategy(ctx, probabilityStrategy, probabilityChannels, options...)
	}
	channels := toSelectableChannelsWithWeightByFreqRatio(priorityChannelsWithFreqRatio)
	return newByStrategy(ctx, strategy, channels, options...)
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

func toSelectableChannelsWithWeightByFreqRatio[T any](
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T]) []selectable.ChannelWithWeight[T, int] {
	res := make([]selectable.ChannelWithWeight[T, int], 0, len(priorityChannelsWithFreqRatio))
	for _, q := range priorityChannelsWithFreqRatio {
		priorityChannel := q.PriorityChannel()
		res = append(res, asSelectableChannelWithWeight(priorityChannel, q.Name(), q.FreqRatio()))
	}
	return res
}

func toProbabilitySelectableChannelsWithWeightByFreqRatio[T any](
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T]) []selectable.ChannelWithWeight[T, float64] {
	res := make([]selectable.ChannelWithWeight[T, float64], 0, len(priorityChannelsWithFreqRatio))
	totalSum := 0.0
	for _, c := range priorityChannelsWithFreqRatio {
		totalSum += float64(c.FreqRatio())
	}
	accSum := 0.0
	for i, c := range priorityChannelsWithFreqRatio {
		var cprob float64
		if i != len(priorityChannelsWithFreqRatio)-1 {
			cprob = float64(c.FreqRatio()) / totalSum
			accSum = accSum + cprob
		} else {
			cprob = 1.0 - accSum
		}
		res = append(res, asSelectableChannelWithWeight(
			c.PriorityChannel(), c.Name(), cprob))
	}
	return res
}
