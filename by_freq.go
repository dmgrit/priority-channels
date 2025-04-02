package priority_channels

import (
	"context"
	"sync"

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

func ProcessByFrequencyRatioWithGoroutines[T any](ctx context.Context,
	channelsWithFreqRatios []channels.ChannelWithFreqRatio[T],
	fnCallback func(msg T, channelName string, status ReceiveStatus)) error {
	if err := validateInputChannels(convertChannelsWithFreqRatioToChannels(channelsWithFreqRatios)); err != nil {
		return err
	}
	var wg sync.WaitGroup
	for i := range channelsWithFreqRatios {
		var closeChannelOnce sync.Once
		for j := 0; j < channelsWithFreqRatios[i].FreqRatio(); j++ {
			wg.Add(1)
			go func(c channels.ChannelWithFreqRatio[T]) {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case msg, ok := <-c.MsgsC():
						if !ok {
							closeChannelOnce.Do(func() {
								fnCallback(getZero[T](), c.ChannelName(), ReceiveChannelClosed)
							})
							return
						}
						fnCallback(msg, c.ChannelName(), ReceiveSuccess)
					}
				}
			}(channelsWithFreqRatios[i])
		}
	}
	go func() {
		wg.Wait()
		select {
		case <-ctx.Done():
			return
		default:
			fnCallback(getZero[T](), "", ReceiveNoOpenChannels)
		}
	}()
	go func() {
		<-ctx.Done()
		fnCallback(getZero[T](), "", ReceiveContextCancelled)
	}()
	return nil
}
