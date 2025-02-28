package priority_channels_test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	pc "github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

type Msg struct {
	Body string
}

func TestProcessMessagesByFrequencyRatio(t *testing.T) {
	msgsChannels := make([]chan *Msg, 4)
	msgsChannels[0] = make(chan *Msg, 15)
	msgsChannels[1] = make(chan *Msg, 15)
	msgsChannels[2] = make(chan *Msg, 15)
	msgsChannels[3] = make(chan *Msg, 15)

	channels := []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithFreqRatio("Priority-5", msgsChannels[1], 5),
		channels.NewChannelWithFreqRatio("Priority-10", msgsChannels[2], 10),
		channels.NewChannelWithFreqRatio("Priority-1000", msgsChannels[3], 1000),
	}

	for i := 0; i <= 2; i++ {
		for j := 1; j <= 15; j++ {
			msgsChannels[i] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[i].ChannelName(), j)}
		}
	}
	msgsChannels[3] <- &Msg{Body: "Priority-1000 Msg-1"}

	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		results = append(results, msg)
	}
	ctx, cancel := context.WithCancel(context.Background())

	priorityChannel, err := pc.NewByFrequencyRatio(ctx, channels)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	go pc.ProcessPriorityChannelMessages(priorityChannel, msgProcessor)

	time.Sleep(3 * time.Second)
	cancel()

	expectedResults := []*Msg{
		{Body: "Priority-1000 Msg-1"},
		{Body: "Priority-10 Msg-1"},
		{Body: "Priority-10 Msg-2"},
		{Body: "Priority-10 Msg-3"},
		{Body: "Priority-10 Msg-4"},
		{Body: "Priority-10 Msg-5"},
		{Body: "Priority-10 Msg-6"},
		{Body: "Priority-10 Msg-7"},
		{Body: "Priority-10 Msg-8"},
		{Body: "Priority-10 Msg-9"},
		{Body: "Priority-10 Msg-10"},
		{Body: "Priority-5 Msg-1"},
		{Body: "Priority-5 Msg-2"},
		{Body: "Priority-5 Msg-3"},
		{Body: "Priority-5 Msg-4"},
		{Body: "Priority-5 Msg-5"},
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-10 Msg-11"},
		{Body: "Priority-10 Msg-12"},
		{Body: "Priority-10 Msg-13"},
		{Body: "Priority-10 Msg-14"},
		{Body: "Priority-10 Msg-15"},
		{Body: "Priority-5 Msg-6"},
		{Body: "Priority-5 Msg-7"},
		{Body: "Priority-5 Msg-8"},
		{Body: "Priority-5 Msg-9"},
		{Body: "Priority-5 Msg-10"},
		{Body: "Priority-1 Msg-2"},
		{Body: "Priority-5 Msg-11"},
		{Body: "Priority-5 Msg-12"},
		{Body: "Priority-5 Msg-13"},
		{Body: "Priority-5 Msg-14"},
		{Body: "Priority-5 Msg-15"},
		{Body: "Priority-1 Msg-3"},
		{Body: "Priority-1 Msg-4"},
		{Body: "Priority-1 Msg-5"},
		{Body: "Priority-1 Msg-6"},
		{Body: "Priority-1 Msg-7"},
		{Body: "Priority-1 Msg-8"},
		{Body: "Priority-1 Msg-9"},
		{Body: "Priority-1 Msg-10"},
		{Body: "Priority-1 Msg-11"},
		{Body: "Priority-1 Msg-12"},
		{Body: "Priority-1 Msg-13"},
		{Body: "Priority-1 Msg-14"},
		{Body: "Priority-1 Msg-15"},
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, but got %d", len(expectedResults), len(results))
	}
	for i := range results {
		if results[i].Body != expectedResults[i].Body {
			t.Errorf("Result %d: Expected message %s, but got %s",
				i, expectedResults[i].Body, results[i].Body)
		}
	}
}

func TestProcessMessagesByFrequencyRatio_TenThousandMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var inputChannels []chan string

	channelsNum := 5
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
	}

	channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio("Channel A", inputChannels[0], 1),
		channels.NewChannelWithFreqRatio("Channel B", inputChannels[1], 2),
		channels.NewChannelWithFreqRatio("Channel C", inputChannels[2], 3),
		channels.NewChannelWithFreqRatio("Channel D", inputChannels[3], 4),
		channels.NewChannelWithFreqRatio("Channel E", inputChannels[4], 5),
	}

	freqTotalSum := 0.0
	for i := 1; i <= channelsNum; i++ {
		freqTotalSum += float64(channelsWithFreqRatio[i-1].FreqRatio())
	}
	expectedRatios := make(map[string]float64)
	for _, ch := range channelsWithFreqRatio {
		expectedRatios[ch.ChannelName()] = float64(ch.FreqRatio()) / freqTotalSum
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

	ch, err := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
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

	for _, channel := range channelsWithFreqRatio {
		expectedRatio := expectedRatios[channel.ChannelName()]
		actualRatio := float64(countPerChannel[channel.ChannelName()]) / float64(totalCount)
		if math.Abs(expectedRatio-actualRatio) > 0.03 {
			t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
				channel.ChannelName(), expectedRatio, actualRatio)
		}
	}
}

func TestProcessMessagesByFrequencyRatio_MessagesInOneOfTheChannelsArriveAfterSomeTime(t *testing.T) {
	msgsChannels := make([]chan *Msg, 3)
	msgsChannels[0] = make(chan *Msg, 7)
	msgsChannels[1] = make(chan *Msg, 7)
	msgsChannels[2] = make(chan *Msg, 7)

	channels := []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithFreqRatio("Priority-2", msgsChannels[1], 2),
		channels.NewChannelWithFreqRatio("Priority-3", msgsChannels[2], 3),
	}

	simulateLongProcessingMsg := "Simulate long processing"
	for j := 1; j <= 5; j++ {
		msgsChannels[0] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[0].ChannelName(), j)}
		suffix := ""
		if j == 5 {
			suffix = " - " + simulateLongProcessingMsg
		}
		msgsChannels[2] <- &Msg{Body: fmt.Sprintf("%s Msg-%d%s", channels[2].ChannelName(), j, suffix)}
	}

	waitForMessagesFromPriority2Chan := make(chan struct{})
	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		if strings.HasSuffix(msg.Body, simulateLongProcessingMsg) {
			<-waitForMessagesFromPriority2Chan
		}
		results = append(results, msg)
	}
	ctx, cancel := context.WithCancel(context.Background())

	priorityChannel, err := pc.NewByFrequencyRatio(ctx, channels)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	go pc.ProcessPriorityChannelMessages(priorityChannel, msgProcessor)

	time.Sleep(1 * time.Second)
	for j := 6; j <= 7; j++ {
		msgsChannels[0] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[0].ChannelName(), j)}
		msgsChannels[2] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[2].ChannelName(), j)}
	}
	msgsChannels[1] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[1].ChannelName(), 1)}
	msgsChannels[1] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[1].ChannelName(), 2)}
	msgsChannels[1] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[1].ChannelName(), 3)}
	waitForMessagesFromPriority2Chan <- struct{}{}

	time.Sleep(3 * time.Second)
	cancel()

	expectedResults := []*Msg{
		{Body: "Priority-3 Msg-1"},
		{Body: "Priority-3 Msg-2"},
		{Body: "Priority-3 Msg-3"},
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-3 Msg-4"},
		{Body: "Priority-3 Msg-5 - Simulate long processing"},
		{Body: "Priority-2 Msg-1"},
		{Body: "Priority-2 Msg-2"},
		{Body: "Priority-3 Msg-6"},
		{Body: "Priority-2 Msg-3"},
		{Body: "Priority-1 Msg-2"},
		{Body: "Priority-3 Msg-7"},
		{Body: "Priority-1 Msg-3"},
		{Body: "Priority-1 Msg-4"},
		{Body: "Priority-1 Msg-5"},
		{Body: "Priority-1 Msg-6"},
		{Body: "Priority-1 Msg-7"},
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, but got %d", len(expectedResults), len(results))
	}
	for i := range results {
		if results[i].Body != expectedResults[i].Body {
			t.Errorf("Result %d: Expected message %s, but got %s",
				i, expectedResults[i].Body, results[i].Body)
		}
	}
}

func TestProcessMessagesByFrequencyRatio_ChannelClosed(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	close(normalPriorityC)

	ch, err := pc.NewByFrequencyRatio(context.Background(), channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	for i := 0; i < 3; i++ {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != pc.ReceiveChannelClosed {
			t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", pc.ReceiveChannelClosed, status)
		}
		if channelName != "Normal Priority" {
			t.Errorf("Expected channel name 'Normal Priority', but got %s", channelName)
		}
		if message != "" {
			t.Errorf("Expected empty message, but got %s", message)
		}
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != pc.ReceiveChannelClosed {
		t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", pc.ReceiveChannelClosed, status)
	}
	if channelName != "Normal Priority" {
		t.Errorf("Expected channel name 'Normal Priority', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByFrequencyRatio_ExitOnDefaultCase(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	ch, err := pc.NewByFrequencyRatio(context.Background(), channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != pc.ReceiveDefaultCase {
		t.Errorf("Expected status ReceiveDefaultCase (%d), but got %d", pc.ReceiveDefaultCase, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByFrequencyRatio_RequestContextCancelled(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	ch, err := pc.NewByFrequencyRatio(context.Background(), channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	message, channelName, status := ch.ReceiveWithContext(ctx)
	if status != pc.ReceiveContextCancelled {
		t.Errorf("Expected status ReceiveContextCancelled (%d), but got %d", pc.ReceiveContextCancelled, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByFrequencyRatio_PriorityChannelContextCancelled(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := pc.NewByFrequencyRatio(ctx, channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(context.Background())
	if status != pc.ReceivePriorityChannelCancelled {
		t.Errorf("Expected status ReceivePriorityChannelCancelled (%d), but got %d", pc.ReceivePriorityChannelCancelled, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestByFrequencyRatioPriorityChannelValidation(t *testing.T) {
	var testCases = []struct {
		Name                   string
		ChannelsWithFreqRatios []channels.ChannelWithFreqRatio[string]
		ExpectedErrorMessage   string
	}{
		{
			Name:                   "No channels",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{},
			ExpectedErrorMessage:   pc.ErrNoChannels.Error(),
		},
		{
			Name: "Empty channel name",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					5),
				channels.NewChannelWithFreqRatio(
					"",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: pc.ErrEmptyChannelName.Error(),
		},
		{
			Name: "Zero frequency ratio value",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					0),
				channels.NewChannelWithFreqRatio(
					"Low Priority Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel 'Normal Messages': frequency ratio must be greater than 0",
		},
		{
			Name: "Negative frequency ratio value",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					-5),
				channels.NewChannelWithFreqRatio(
					"Low Priority Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel 'Normal Messages': frequency ratio must be greater than 0",
		},
		{
			Name: "Duplicate channel name",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					5),
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel name 'Urgent Messages' is used more than once",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			_, err := pc.NewByFrequencyRatio(ctx, tc.ChannelsWithFreqRatios)
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
