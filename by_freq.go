package priority_channels

import (
	"context"
	"errors"
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
	selectableChannels := make([]selectable.Channel[T], 0, len(channelsWithFreqRatios))
	selectableChannelsWeights := make([]int, 0, len(channelsWithFreqRatios))
	for _, c := range channelsWithFreqRatios {
		selectableChannels = append(selectableChannels, selectable.NewFromInputChannel(c.ChannelName(), c.MsgsC()))
		selectableChannelsWeights = append(selectableChannelsWeights, c.FreqRatio())
	}
	return newByStrategy(ctx, strategy, selectableChannels, selectableChannelsWeights, options...)
}

func ProcessByFrequencyRatioWithGoroutines[T any](ctx context.Context,
	channelsWithFreqRatios []channels.ChannelWithFreqRatio[T],
	onMessageReceived func(msg T, channelName string),
	onChannelClosed func(channelName string),
	onProcessingFinished func(reason ExitReason)) error {
	if err := validateInputChannels(convertChannelsWithFreqRatioToChannels(channelsWithFreqRatios)); err != nil {
		return err
	}
	if onMessageReceived == nil {
		return errors.New("onMessageReceived callback is nil")
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
								if onChannelClosed != nil {
									onChannelClosed(c.ChannelName())
								}
							})
							return
						}
						onMessageReceived(msg, c.ChannelName())
					}
				}
			}(channelsWithFreqRatios[i])
		}
	}
	go func() {
		wg.Wait()
		select {
		case <-ctx.Done():
			if onProcessingFinished != nil {
				onProcessingFinished(ContextCanceled)
			}
			return
		default:
			if onProcessingFinished != nil {
				onProcessingFinished(NoReceivablePath)
			}
		}
	}()
	return nil
}
