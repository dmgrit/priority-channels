package priority_channels_test

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies"
)

func TestProcessMessagesByProbability(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var channelsWithProbability []channels.ChannelWithWeight[string, float64]
	var inputChannels []chan string

	channelsNum := 5
	arithmeticSum := 0
	for i := 1; i <= channelsNum; i++ {
		arithmeticSum += i
	}
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
		probability := float64(i) / float64(arithmeticSum)
		channelName := fmt.Sprintf("Probability %.2f", probability)
		channelsWithProbability = append(channelsWithProbability, channels.NewChannelWithWeight[string, float64](
			channelName,
			inputChannels[i-1],
			probability))
	}

	ch, err := priority_channels.NewByStrategy(ctx, strategies.NewByProbability(), channelsWithProbability)
	if err != nil {
		t.Errorf("Failed to create priority channel: %v\n", err)
	}

	for i := 1; i <= channelsNum; i++ {
		go func(i int) {
			for j := 1; j <= 10000; j++ {
				select {
				case <-ctx.Done():
					return
				case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
				}
			}
		}(i)
	}

	totalCount := 0
	countPerChannel := make(map[string]int)
	go func() {
		for {
			_, channel, ok := ch.Receive()
			if !ok {
				return
			}
			totalCount++
			countPerChannel[channel] = countPerChannel[channel] + 1
			if totalCount == 10000 {
				cancel()
				return
			}
		}
	}()

	<-ctx.Done()

	for _, channel := range channelsWithProbability {
		expectedProbability := channel.Weight()
		actualProbability := float64(countPerChannel[channel.ChannelName()]) / float64(totalCount)
		if math.Abs(actualProbability-expectedProbability) > 0.01 {
			t.Errorf("Channel %s: expected messages number by probability %.2f, got %.2f\n",
				channel.ChannelName(), expectedProbability, actualProbability)
		}
	}

}
