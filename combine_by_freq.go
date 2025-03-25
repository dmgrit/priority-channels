package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func CombineByFrequencyRatio[T any](ctx context.Context,
	priorityChannelsWithFreqRatio []PriorityChannelWithFreqRatio[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	channels := toSelectableChannelsWithWeightByFreqRatio(priorityChannelsWithFreqRatio)
	sumFreqRatios := 0
	for _, c := range priorityChannelsWithFreqRatio {
		sumFreqRatios += c.FreqRatio()
	}
	strategy, err := getFrequencyStrategy(levelCombine, pcOptions.frequencyMode, pcOptions.frequencyMethod, sumFreqRatios)
	if err != nil {
		return nil, err
	}
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
