package priority_channels_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	pc "github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func TestProcessMessagesByPriorityWithHighestAlwaysFirst(t *testing.T) {
	msgsChannels := make([]chan *Msg, 4)
	msgsChannels[0] = make(chan *Msg, 15)
	msgsChannels[1] = make(chan *Msg, 15)
	msgsChannels[2] = make(chan *Msg, 15)
	msgsChannels[3] = make(chan *Msg, 15)

	channels := []channels.ChannelWithPriority[*Msg]{
		channels.NewChannelWithPriority("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithPriority("Priority-5", msgsChannels[1], 5),
		channels.NewChannelWithPriority("Priority-10", msgsChannels[2], 10),
		channels.NewChannelWithPriority("Priority-1000", msgsChannels[3], 1000),
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

	priorityChannel, err := pc.NewByHighestAlwaysFirst(ctx, channels)
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
		{Body: "Priority-10 Msg-11"},
		{Body: "Priority-10 Msg-12"},
		{Body: "Priority-10 Msg-13"},
		{Body: "Priority-10 Msg-14"},
		{Body: "Priority-10 Msg-15"},
		{Body: "Priority-5 Msg-1"},
		{Body: "Priority-5 Msg-2"},
		{Body: "Priority-5 Msg-3"},
		{Body: "Priority-5 Msg-4"},
		{Body: "Priority-5 Msg-5"},
		{Body: "Priority-5 Msg-6"},
		{Body: "Priority-5 Msg-7"},
		{Body: "Priority-5 Msg-8"},
		{Body: "Priority-5 Msg-9"},
		{Body: "Priority-5 Msg-10"},
		{Body: "Priority-5 Msg-11"},
		{Body: "Priority-5 Msg-12"},
		{Body: "Priority-5 Msg-13"},
		{Body: "Priority-5 Msg-14"},
		{Body: "Priority-5 Msg-15"},
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-1 Msg-2"},
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

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_CustomWaitInterval(t *testing.T) {
	ctx := context.Background()

	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	// sending messages to individual channels
	go func() {
		for i := 1; i <= 5; i++ {
			highPriorityC <- fmt.Sprintf("high priority message %d", i)
			// Simulating high priority messages arriving at a slower rate
			time.Sleep(500 * time.Microsecond)
		}
	}()
	go func() {
		for i := 1; i <= 5; i++ {
			normalPriorityC <- fmt.Sprintf("normal priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 5; i++ {
			lowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
	}()

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithPriority(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Low Priority",
			lowPriorityC,
			3),
	}
	var results []string
	msgProcessor := func(_ context.Context, msg string, channelName string) {
		//fmt.Printf("%s: %s\n", channelName, msg)
		results = append(results, msg)
	}
	ctx, cancel := context.WithCancel(context.Background())

	priorityChannel, err := pc.NewByHighestAlwaysFirst(ctx, channelsWithPriority, pc.ChannelWaitInterval(1*time.Millisecond))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	go pc.ProcessPriorityChannelMessages(priorityChannel, msgProcessor)

	time.Sleep(1 * time.Second)
	cancel()

	expectedResults := []string{
		"high priority message 1",
		"high priority message 2",
		"high priority message 3",
		"high priority message 4",
		"high priority message 5",
		"normal priority message 1",
		"normal priority message 2",
		"normal priority message 3",
		"normal priority message 4",
		"normal priority message 5",
		"low priority message 1",
		"low priority message 2",
		"low priority message 3",
		"low priority message 4",
		"low priority message 5",
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, but got %d", len(expectedResults), len(results))
	}
	for i := range results {
		if results[i] != expectedResults[i] {
			t.Errorf("Result %d: Expected message %s, but got %s",
				i, expectedResults[i], results[i])
		}
	}
}

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_AutoDisableClosedChannels(t *testing.T) {
	ctx := context.Background()

	urgentMessagesC := make(chan string)
	highPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	// sending messages to individual channels
	go func() {
		for i := 1; i <= 20; i++ {
			highPriorityC <- fmt.Sprintf("high priority message %d", i)
		}
		close(highPriorityC)
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			lowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
		close(lowPriorityC)
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			urgentMessagesC <- fmt.Sprintf("urgent message %d", i)
		}
		close(urgentMessagesC)
	}()

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"High Priority",
			highPriorityC,
			8),
		channels.NewChannelWithPriority(
			"Low Priority",
			lowPriorityC,
			3),
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentMessagesC,
			10),
	}
	ch, err := pc.NewByHighestAlwaysFirst(ctx, channelsWithPriority, pc.AutoDisableClosedChannels())
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	receivedMessagesCount := 0
	for {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != pc.ReceiveSuccess {
			if receivedMessagesCount != 60 {
				t.Errorf("Expected to receive 60 messages, but got %d", receivedMessagesCount)
			}
			if status != pc.ReceiveNoOpenChannels {
				t.Errorf("Expected to receive 'no open channels' status on closure (%v), but got %v",
					pc.ReceiveNoOpenChannels, status)
			}
			break
		}
		receivedMessagesCount++
		fmt.Printf("%s: %s\n", channelName, message)
		time.Sleep(10 * time.Millisecond)
	}
}

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_MessagesInOneOfTheChannelsArriveAfterSomeTime(t *testing.T) {
	msgsChannels := make([]chan *Msg, 3)
	msgsChannels[0] = make(chan *Msg, 7)
	msgsChannels[1] = make(chan *Msg, 7)
	msgsChannels[2] = make(chan *Msg, 7)

	channels := []channels.ChannelWithPriority[*Msg]{
		channels.NewChannelWithPriority("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithPriority("Priority-2", msgsChannels[1], 2),
		channels.NewChannelWithPriority("Priority-3", msgsChannels[2], 3),
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

	priorityChannel, err := pc.NewByHighestAlwaysFirst(ctx, channels)
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
		{Body: "Priority-3 Msg-4"},
		{Body: "Priority-3 Msg-5 - Simulate long processing"},
		{Body: "Priority-3 Msg-6"},
		{Body: "Priority-3 Msg-7"},
		{Body: "Priority-2 Msg-1"},
		{Body: "Priority-2 Msg-2"},
		{Body: "Priority-2 Msg-3"},
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-1 Msg-2"},
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

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_ChannelClose(t *testing.T) {
	urgentC := make(chan string)
	normalC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentC,
			10),
		channels.NewChannelWithPriority(
			"Normal Messages",
			normalC,
			5),
		channels.NewChannelWithPriority(
			"Low Priority Messages",
			lowPriorityC,
			1),
	}

	close(normalC)

	ch, err := pc.NewByHighestAlwaysFirst(context.Background(), channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	for i := 0; i < 3; i++ {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != pc.ReceiveChannelClosed {
			t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", pc.ReceiveChannelClosed, status)
		}
		if channelName != "Normal Messages" {
			t.Errorf("Expected channel name 'Normal Messages', but got %s", channelName)
		}
		if message != "" {
			t.Errorf("Expected empty message, but got %s", message)
		}
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != pc.ReceiveChannelClosed {
		t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", pc.ReceiveChannelClosed, status)
	}
	if channelName != "Normal Messages" {
		t.Errorf("Expected channel name 'Normal Messages', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_ExitOnDefaultCase(t *testing.T) {
	urgentC := make(chan string)
	normalC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentC,
			10),
		channels.NewChannelWithPriority(
			"Normal Messages",
			normalC,
			5),
		channels.NewChannelWithPriority(
			"Low Priority Messages",
			lowPriorityC,
			1),
	}

	ch, err := pc.NewByHighestAlwaysFirst(context.Background(), channelsWithPriority)
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

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_RequestContextCancelled(t *testing.T) {
	urgentC := make(chan string)
	normalC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentC,
			10),
		channels.NewChannelWithPriority(
			"Normal Messages",
			normalC,
			5),
		channels.NewChannelWithPriority(
			"Low Priority Messages",
			lowPriorityC,
			1),
	}

	ch, err := pc.NewByHighestAlwaysFirst(context.Background(), channelsWithPriority)
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

func TestProcessMessagesByPriorityWithHighestAlwaysFirst_PriorityChannelContextCancelled(t *testing.T) {
	urgentC := make(chan string)
	normalC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentC,
			10),
		channels.NewChannelWithPriority(
			"Normal Messages",
			normalC,
			5),
		channels.NewChannelWithPriority(
			"Low Priority Messages",
			lowPriorityC,
			1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch, err := pc.NewByHighestAlwaysFirst(ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel initialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(context.Background())
	if status != pc.ReceivePriorityChannelClosed {
		t.Errorf("Expected status ReceivePriorityChannelClosed (%d), but got %d", pc.ReceivePriorityChannelClosed, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestHighestAlwaysFirstPriorityChannelValidation(t *testing.T) {
	var testCases = []struct {
		Name                   string
		ChannelsWithPriorities []channels.ChannelWithPriority[string]
		ExpectedErrorMessage   string
	}{
		{
			Name:                   "No channels",
			ChannelsWithPriorities: []channels.ChannelWithPriority[string]{},
			ExpectedErrorMessage:   pc.ErrNoChannels.Error(),
		},
		{
			Name: "Empty channel name",
			ChannelsWithPriorities: []channels.ChannelWithPriority[string]{
				channels.NewChannelWithPriority(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithPriority(
					"Normal Messages",
					make(chan string),
					5),
				channels.NewChannelWithPriority(
					"",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: pc.ErrEmptyChannelName.Error(),
		},
		{
			Name: "Zero priority value - No error is expected",
			ChannelsWithPriorities: []channels.ChannelWithPriority[string]{
				channels.NewChannelWithPriority(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithPriority(
					"Normal Messages",
					make(chan string),
					0),
				channels.NewChannelWithPriority(
					"Low Priority Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "",
		},
		{
			Name: "Negative priority value",
			ChannelsWithPriorities: []channels.ChannelWithPriority[string]{
				channels.NewChannelWithPriority(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithPriority(
					"Normal Messages",
					make(chan string),
					-5),
				channels.NewChannelWithPriority(
					"Low Priority Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel 'Normal Messages': priority cannot be negative",
		},
		{
			Name: "Duplicate channel name",
			ChannelsWithPriorities: []channels.ChannelWithPriority[string]{
				channels.NewChannelWithPriority(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithPriority(
					"Normal Messages",
					make(chan string),
					5),
				channels.NewChannelWithPriority(
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
			_, err := pc.NewByHighestAlwaysFirst(ctx, tc.ChannelsWithPriorities)
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
				t.Errorf("Expected error %s, but got: %v", tc.ExpectedErrorMessage, err)
			}
		})
	}
}
