package priority_channels_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
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

	ch, err := priority_channels.NewByStrategy(ctx, priority_strategies.NewByProbability(), channelsWithProbability)
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
		if math.Abs(actualProbability-expectedProbability) > 0.03 {
			t.Errorf("Channel %s: expected messages number by probability %.2f, got %.2f\n",
				channel.ChannelName(), expectedProbability, actualProbability)
		}
	}
}

func TestProcessMessagesByProbability_AutoDisableClosedChannels(t *testing.T) {
	ctx := context.Background()

	urgentMessagesC := make(chan string)
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	// sending messages to individual channels
	go func() {
		for i := 1; i <= 50; i++ {
			highPriorityC <- fmt.Sprintf("high priority message %d", i)
		}
		close(highPriorityC)
	}()
	go func() {
		for i := 1; i <= 50; i++ {
			normalPriorityC <- fmt.Sprintf("normal priority message %d", i)
		}
		close(normalPriorityC)
	}()
	go func() {
		for i := 1; i <= 50; i++ {
			lowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
		close(lowPriorityC)
	}()
	go func() {
		for i := 1; i <= 50; i++ {
			urgentMessagesC <- fmt.Sprintf("urgent message %d", i)
		}
		close(urgentMessagesC)
	}()

	channelsWithProbability := []channels.ChannelWithWeight[string, float64]{
		channels.NewChannelWithWeight[string, float64](
			"High Priority",
			highPriorityC,
			0.3),
		channels.NewChannelWithWeight(
			"Normal Priority",
			normalPriorityC,
			0.2),
		channels.NewChannelWithWeight(
			"Low Priority",
			lowPriorityC,
			0.1),
		channels.NewChannelWithWeight(
			"Urgent Messages",
			urgentMessagesC,
			0.4),
	}
	ch, err := priority_channels.NewByStrategy(ctx, priority_strategies.NewByProbability(), channelsWithProbability,
		priority_channels.AutoDisableClosedChannels())
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	receivedMessagesCount := 0
	for {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != priority_channels.ReceiveSuccess {
			if receivedMessagesCount != 200 {
				t.Errorf("Expected to receive 200 messages, but got %d", receivedMessagesCount)
			}
			if status != priority_channels.ReceiveNoOpenChannels {
				t.Errorf("Expected to receive 'no open channels' status on closure (%v), but got %v",
					priority_channels.ReceiveNoOpenChannels, status)
			}
			break
		}
		receivedMessagesCount++
		fmt.Printf("%s: %s\n", channelName, message)
		time.Sleep(10 * time.Millisecond)
	}
}

func TestByProbabilityPriorityChannelValidation(t *testing.T) {
	var testCases = []struct {
		Name                      string
		ChannelsWithProbabilities []channels.ChannelWithWeight[string, float64]
		ExpectedErrorMessage      string
	}{
		{
			Name: "Probability less than 0",
			ChannelsWithProbabilities: []channels.ChannelWithWeight[string, float64]{
				channels.NewChannelWithWeight(
					"Channel 1",
					make(chan string),
					0.5),
				channels.NewChannelWithWeight(
					"Channel 2",
					make(chan string),
					-0.5),
			},
			ExpectedErrorMessage: "channel 'Channel 2': probability must be between 0 and 1 (exclusive)",
		},
		{
			Name: "Probability equals to 0",
			ChannelsWithProbabilities: []channels.ChannelWithWeight[string, float64]{
				channels.NewChannelWithWeight(
					"Channel 1",
					make(chan string),
					0.0),
				channels.NewChannelWithWeight(
					"Channel 2",
					make(chan string),
					0.5),
			},
			ExpectedErrorMessage: "channel 'Channel 1': probability must be between 0 and 1 (exclusive)",
		},
		{
			Name: "Probability equals to 1",
			ChannelsWithProbabilities: []channels.ChannelWithWeight[string, float64]{
				channels.NewChannelWithWeight(
					"Channel 1",
					make(chan string),
					1.0),
				channels.NewChannelWithWeight(
					"Channel 2",
					make(chan string),
					0.5),
			},
			ExpectedErrorMessage: "channel 'Channel 1': probability must be between 0 and 1 (exclusive)",
		},
		{
			Name: "Probabilities sum not equal to 1",
			ChannelsWithProbabilities: []channels.ChannelWithWeight[string, float64]{
				channels.NewChannelWithWeight(
					"Channel 1",
					make(chan string),
					0.5),
				channels.NewChannelWithWeight(
					"Channel 2",
					make(chan string),
					0.4),
			},
			ExpectedErrorMessage: "sum of probabilities must be 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			_, err := priority_channels.NewByStrategy(ctx, priority_strategies.NewByProbability(), tc.ChannelsWithProbabilities)
			if tc.ExpectedErrorMessage == "" {
				if err != nil {
					t.Fatalf("Unexpected validation error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Expected validation error")
			}
			if err.Error() != tc.ExpectedErrorMessage {
				t.Errorf("Expected error %v, but got: %v", tc.ExpectedErrorMessage, err)
			}
		})
	}
}
