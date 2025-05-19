package priority_channels_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	msgsChannels := make([]chan *Msg, 4)
	msgsChannels[0] = make(chan *Msg, 15)
	msgsChannels[1] = make(chan *Msg, 15)
	msgsChannels[2] = make(chan *Msg, 15)
	msgsChannels[3] = make(chan *Msg, 15)

	options := priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully)
	group1Priority1Channel, err := priority_channels.NewByFrequencyRatio[*Msg](ctx, []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithFreqRatio("Priority-5", msgsChannels[1], 5),
	}, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	group2Priority2Channel, err := priority_channels.WrapAsPriorityChannel[*Msg](ctx,
		"Priority-10", msgsChannels[2])
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	group3PriorityChannel, err := priority_channels.WrapAsPriorityChannel[*Msg](ctx,
		"Priority-1000", msgsChannels[3])
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channels := []priority_channels.PriorityChannelWithPriority[*Msg]{
		priority_channels.NewPriorityChannelWithPriority[*Msg](
			"Group 1",
			group1Priority1Channel,
			1),
		priority_channels.NewPriorityChannelWithPriority[*Msg](
			"Group 2",
			group2Priority2Channel,
			10),
		priority_channels.NewPriorityChannelWithPriority[*Msg](
			"Group 3",
			group3PriorityChannel,
			1000),
	}

	channelNames := []string{"Priority-1", "Priority-5", "Priority-10", "Priority-1000"}

	for i := 0; i <= 2; i++ {
		for j := 1; j <= 15; j++ {
			msgsChannels[i] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channelNames[i], j)}
		}
	}
	msgsChannels[3] <- &Msg{Body: "Priority-1000 Msg-1"}

	done := make(chan struct{})
	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		results = append(results, msg)
		if len(results) == 46 {
			done <- struct{}{}
		}
	}

	priorityChannel, err := priority_channels.CombineByHighestAlwaysFirst(ctx, channels)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	processingDone := make(chan priority_channels.ExitReason)
	go priority_channels.ProcessPriorityChannelMessages(priorityChannel, msgProcessor, processingDone)

	<-done
	cancel()
	<-processingDone

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
		{Body: "Priority-1 Msg-1"},
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

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_MessagesInOneOfTheChannelsArriveAfterSomeTime(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	msgsChannels := make([]chan *Msg, 3)
	msgsChannels[0] = make(chan *Msg, 7)
	msgsChannels[1] = make(chan *Msg, 7)
	msgsChannels[2] = make(chan *Msg, 7)

	options := priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully)
	group1PriorityChannel, err := priority_channels.NewByFrequencyRatio[*Msg](ctx, []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithFreqRatio("Priority-2", msgsChannels[1], 2),
	}, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	group2PriorityChannel, err := priority_channels.NewByFrequencyRatio[*Msg](ctx, []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-3", msgsChannels[2], 1),
	}, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channels := []priority_channels.PriorityChannelWithPriority[*Msg]{
		priority_channels.NewPriorityChannelWithPriority[*Msg](
			"Group 1",
			group1PriorityChannel,
			1),
		priority_channels.NewPriorityChannelWithPriority[*Msg](
			"Group 2",
			group2PriorityChannel,
			2),
	}

	simulateLongProcessingMsg := "Simulate long processing"
	for j := 1; j <= 5; j++ {
		msgsChannels[0] <- &Msg{Body: fmt.Sprintf("Priority-1 Msg-%d", j)}
		suffix := ""
		if j == 5 {
			suffix = " - " + simulateLongProcessingMsg
		}
		msgsChannels[2] <- &Msg{Body: fmt.Sprintf("Priority-3 Msg-%d%s", j, suffix)}
	}

	waitForMessagesFromPriority2Chan := make(chan struct{})
	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		if strings.HasSuffix(msg.Body, simulateLongProcessingMsg) {
			<-waitForMessagesFromPriority2Chan
		}
		results = append(results, msg)
	}

	priorityChannel, err := priority_channels.CombineByHighestAlwaysFirst(ctx, channels)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	done := make(chan priority_channels.ExitReason)
	go priority_channels.ProcessPriorityChannelMessages(priorityChannel, msgProcessor, done)

	time.Sleep(1 * time.Second)
	for j := 6; j <= 7; j++ {
		msgsChannels[0] <- &Msg{Body: fmt.Sprintf("Priority-1 Msg-%d", j)}
		msgsChannels[2] <- &Msg{Body: fmt.Sprintf("Priority-3 Msg-%d", j)}
	}
	for j := 1; j <= 7; j++ {
		msgsChannels[1] <- &Msg{Body: fmt.Sprintf("Priority-2 Msg-%d", j)}
	}
	waitForMessagesFromPriority2Chan <- struct{}{}

	time.Sleep(3 * time.Second)
	cancel()
	<-done

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
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-2 Msg-3"},
		{Body: "Priority-2 Msg-4"},
		{Body: "Priority-1 Msg-2"},
		{Body: "Priority-2 Msg-5"},
		{Body: "Priority-2 Msg-6"},
		{Body: "Priority-1 Msg-3"},
		{Body: "Priority-2 Msg-7"},
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

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_ChannelClosed(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithPriority("Free User",
			freeUserPriorityChannel,
			1),
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst[string](ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	close(freeUserHighPriorityC)

	for i := 0; i < 3; i++ {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != priority_channels.ReceiveChannelClosed {
			t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", priority_channels.ReceiveChannelClosed, status)
		}
		if channelName != "Free User - High Priority" {
			t.Errorf("Expected channel name 'Free User - High Priority', but got %s", channelName)
		}
		if message != "" {
			t.Errorf("Expected empty message, but got %s", message)
		}
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != priority_channels.ReceiveChannelClosed {
		t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", priority_channels.ReceiveChannelClosed, status)
	}
	if channelName != "Free User - High Priority" {
		t.Errorf("Expected channel name 'Free User - High Priority', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_ExitOnDefaultCase(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithPriority("Free User",
			freeUserPriorityChannel,
			1),
	}
	ch, err := priority_channels.CombineByHighestAlwaysFirst[string](ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != priority_channels.ReceiveDefaultCase {
		t.Errorf("Expected status ReceiveDefaultCase (%d), but got %d", priority_channels.ReceiveDefaultCase, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_RequestContextCanceled(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithPriority("Free User",
			freeUserPriorityChannel,
			1),
	}
	ch, err := priority_channels.CombineByHighestAlwaysFirst[string](ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()

	message, channelName, status := ch.ReceiveWithContext(ctxWithCancel)
	if status != priority_channels.ReceiveContextCanceled {
		t.Errorf("Expected status ReceiveContextCanceled (%d), but got %d", priority_channels.ReceiveContextCanceled, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_PriorityChannelContextCanceled(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithPriority("Free User",
			freeUserPriorityChannel,
			1),
	}

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()
	ch, err := priority_channels.CombineByHighestAlwaysFirst[string](ctxWithCancel, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(ctx)
	if status != priority_channels.ReceivePriorityChannelClosed {
		t.Errorf("Expected status ReceivePriorityChannelClosed (%d), but got %d", priority_channels.ReceivePriorityChannelClosed, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_InnerPriorityChannelContextCanceled(t *testing.T) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()

	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctxWithCancel, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithPriority("Free User",
			freeUserPriorityChannel,
			1),
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst[string](ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(ctx)
	if status != priority_channels.ReceivePriorityChannelClosed {
		t.Errorf("Expected status ReceivePriorityChannelClosed (%d), but got %d", priority_channels.ReceivePriorityChannelClosed, status)
	}
	if channelName != "Free User" {
		t.Errorf("Expected channel name 'Free User', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_DeepHierarchy_InnerPriorityChannelContextCanceled(t *testing.T) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()

	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)
	urgentMessagesC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctxWithCancel, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	priorityChannelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Free User",
			freeUserPriorityChannel,
			1),
	}
	combinedUsersAndMessageTypesPriorityChannel, err := priority_channels.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, "Urgent Messages", urgentMessagesC)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst(ctx, []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority(
			"Combined Users and Message Types",
			combinedUsersAndMessageTypesPriorityChannel,
			1),
		priority_channels.NewPriorityChannelWithPriority(
			"Urgent Messages",
			urgentMessagesPriorityChannel,
			100),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(ctx)
	if status != priority_channels.ReceivePriorityChannelClosed {
		t.Errorf("Expected status ReceivePriorityChannelClosed (%d), but got %d", priority_channels.ReceivePriorityChannelClosed, status)
	}
	if channelName != "Free User" {
		t.Errorf("Expected channel name 'Free User', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByPriorityAmongFreqRatioChannelGroups_DeepHierarchy_ChannelClosed(t *testing.T) {
	ctx := context.Background()

	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)
	urgentMessagesC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	priorityChannelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Free User",
			freeUserPriorityChannel,
			1),
	}
	combinedUsersAndMessageTypesPriorityChannel, err := priority_channels.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, "Urgent Messages", urgentMessagesC)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst(ctx, []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority(
			"Combined Users and Message Types",
			combinedUsersAndMessageTypesPriorityChannel,
			1),
		priority_channels.NewPriorityChannelWithPriority(
			"Urgent Messages",
			urgentMessagesPriorityChannel,
			100),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	close(freeUserHighPriorityC)

	message, channelName, status := ch.ReceiveWithContext(context.Background())
	if status != priority_channels.ReceiveChannelClosed {
		t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", priority_channels.ReceiveChannelClosed, status)
	}
	if channelName != "Free User - High Priority" {
		t.Errorf("Expected channel name 'Free User - High Priority', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestCombineHighestAlwaysFirstPriorityChannelValidation(t *testing.T) {
	var testCases = []struct {
		Name                   string
		ChannelsWithPriorities []channels.ChannelWithPriority[string]
		ExpectedErrorMessage   string
	}{
		{
			Name:                   "No channels",
			ChannelsWithPriorities: []channels.ChannelWithPriority[string]{},
			ExpectedErrorMessage:   priority_channels.ErrNoChannels.Error(),
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
			ExpectedErrorMessage: priority_channels.ErrEmptyChannelName.Error(),
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

			priorityChannels := make([]priority_channels.PriorityChannelWithPriority[string], 0, len(tc.ChannelsWithPriorities))
			for i, ch := range tc.ChannelsWithPriorities {
				pch, err := priority_channels.WrapAsPriorityChannel(ctx, fmt.Sprintf("channel %d", i), make(chan string)) //ch.MsgsC())
				if err != nil {
					t.Fatalf("Unexpected error on wrapping as priority channel: %v", err)
				}
				priorityChannels = append(priorityChannels, priority_channels.NewPriorityChannelWithPriority(
					ch.ChannelName(), pch, ch.Priority()))
			}

			_, err := priority_channels.CombineByHighestAlwaysFirst(ctx, priorityChannels)
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
