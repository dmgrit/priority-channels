package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func NewByFrequencyRatio[T any](ctx context.Context,
	channelsWithFreqRatios []channels.ChannelWithFreqRatio[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	strategy, probabilityStrategy := getFrequencyStrategy(options...)
	if probabilityStrategy != nil {
		probabilityChannels := toProbabilitySelectableChannels(channelsWithFreqRatios)
		return newByStrategy(ctx, probabilityStrategy, probabilityChannels, options...)
	}
	selectableChannels := make([]selectable.ChannelWithWeight[T, int], 0, len(channelsWithFreqRatios))
	for _, c := range channelsWithFreqRatios {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithWeight(
			channels.NewChannelWithWeight[T, int](c.ChannelName(), c.MsgsC(), c.FreqRatio()),
		))
	}
	return newByStrategy(ctx, strategy, selectableChannels, options...)
}

func getFrequencyStrategy(options ...func(*PriorityChannelOptions)) (PrioritizationStrategy[int], PrioritizationStrategy[float64]) {
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	switch {
	case pcOptions.frequencyMethod == ProbabilisticByMultipleRandCalls:
		return nil, strategies.NewByProbability()
	case pcOptions.frequencyMethod == ProbabilisticByCaseDuplication:
		return frequency_strategies.NewProbabilisticByCaseDuplication(), nil
	case pcOptions.frequencyMethod == StrictOrderFully:
		return frequency_strategies.NewWithStrictOrderFully(), nil
	case pcOptions.frequencyMethod == StrictOrderAcrossCycles:
		return frequency_strategies.NewWithStrictOrderAcrossCycles(), nil
	default:
		return frequency_strategies.NewProbabilisticByCaseDuplication(), nil
	}
}

func toProbabilitySelectableChannels[T any](channelsWithFreqRatios []channels.ChannelWithFreqRatio[T]) []selectable.ChannelWithWeight[T, float64] {
	res := make([]selectable.ChannelWithWeight[T, float64], 0, len(channelsWithFreqRatios))
	totalSum := 0.0
	for _, c := range channelsWithFreqRatios {
		totalSum += float64(c.FreqRatio())
	}
	accSum := 0.0
	for i, c := range channelsWithFreqRatios {
		var cprob float64
		if i != len(channelsWithFreqRatios)-1 {
			cprob = float64(c.FreqRatio()) / totalSum
			accSum = accSum + cprob
		} else {
			cprob = 1 - accSum
		}
		res = append(res, selectable.NewChannelWithWeight(
			channels.NewChannelWithWeight[T, float64](c.ChannelName(), c.MsgsC(), cprob),
		))
	}
	return res
}
