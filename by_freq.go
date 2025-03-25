package priority_channels

import (
	"context"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/internal/selectable"
)

func NewByFrequencyRatio[T any](ctx context.Context,
	channelsWithFreqRatios []channels.ChannelWithFreqRatio[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	sumFreqRatios := 0
	for _, c := range channelsWithFreqRatios {
		sumFreqRatios += c.FreqRatio()
	}
	strategy, err := getFrequencyStrategy(levelNew, pcOptions.frequencyMode, pcOptions.frequencyMethod, sumFreqRatios)
	if err != nil {
		return nil, err
	}
	selectableChannels := make([]selectable.ChannelWithWeight[T, int], 0, len(channelsWithFreqRatios))
	for _, c := range channelsWithFreqRatios {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithWeight(
			channels.NewChannelWithWeight[T, int](c.ChannelName(), c.MsgsC(), c.FreqRatio()),
		))
	}
	return newByStrategy(ctx, strategy, selectableChannels, options...)
}
