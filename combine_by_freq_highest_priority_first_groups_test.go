package priority_channels_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func TestProcessMessagesByFreqRatioAmongHighestFirstChannelGroups(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	msgsChannels := make([]chan *Msg, 4)
	msgsChannels[0] = make(chan *Msg, 15)
	msgsChannels[1] = make(chan *Msg, 15)
	msgsChannels[2] = make(chan *Msg, 15)
	msgsChannels[3] = make(chan *Msg, 15)

	group1PriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[*Msg]{
		channels.NewChannelWithPriority("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithPriority("Priority-5", msgsChannels[1], 5),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	group2PriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[*Msg]{
		channels.NewChannelWithPriority("Priority-10", msgsChannels[2], 10),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	group3PriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[*Msg]{
		channels.NewChannelWithPriority("Priority-1000", msgsChannels[3], 1000),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channels := []priority_channels.PriorityChannelWithFreqRatio[*Msg]{
		priority_channels.NewPriorityChannelWithFreqRatio("Group 1",
			group1PriorityChannel,
			1),
		priority_channels.NewPriorityChannelWithFreqRatio("Group 2",
			group2PriorityChannel,
			5,
		),
		priority_channels.NewPriorityChannelWithFreqRatio("Group 3",
			group3PriorityChannel,
			10),
	}

	channelNames := []string{"Priority-1", "Priority-5", "Priority-10", "Priority-1000"}

	for i := 0; i <= 2; i++ {
		for j := 1; j <= 15; j++ {
			msgsChannels[i] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channelNames[i], j)}
		}
	}
	msgsChannels[3] <- &Msg{Body: "Priority-1000 Msg-1"}

	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		results = append(results, msg)
	}

	options := priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully)
	priorityChannel, err := priority_channels.CombineByFrequencyRatio(ctx, channels, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	done := make(chan priority_channels.ExitReason)
	go priority_channels.ProcessPriorityChannelMessages(priorityChannel, msgProcessor, done)

	time.Sleep(3 * time.Second)
	cancel()
	<-done

	expectedResults := []*Msg{
		{Body: "Priority-1000 Msg-1"},
		{Body: "Priority-10 Msg-1"},
		{Body: "Priority-10 Msg-2"},
		{Body: "Priority-10 Msg-3"},
		{Body: "Priority-10 Msg-4"},
		{Body: "Priority-10 Msg-5"},
		{Body: "Priority-5 Msg-1"},
		{Body: "Priority-10 Msg-6"},
		{Body: "Priority-10 Msg-7"},
		{Body: "Priority-10 Msg-8"},
		{Body: "Priority-10 Msg-9"},
		{Body: "Priority-10 Msg-10"},
		{Body: "Priority-5 Msg-2"},
		{Body: "Priority-10 Msg-11"},
		{Body: "Priority-10 Msg-12"},
		{Body: "Priority-10 Msg-13"},
		{Body: "Priority-10 Msg-14"},
		{Body: "Priority-10 Msg-15"},
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

func TestProcessMessagesByFreqRatioAmongHighestFirstChannelGroups_ChannelClosed(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Free User",
			freeUserPriorityChannel,
			1),
	}
	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio)
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

func TestProcessMessagesByFreqRatioAmongHighestFirstChannelGroups_ExitOnDefaultCase(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Free User",
			freeUserPriorityChannel,
			1),
	}
	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio)
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

func TestProcessMessagesByFreqRatioAmongHighestFirstChannelGroups_RequestContextCanceled(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Free User",
			freeUserPriorityChannel,
			1),
	}
	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio)
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

func TestProcessMessagesByFreqRatioAmongHighestFirstChannelGroups_PriorityChannelContextCanceled(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Free User",
			freeUserPriorityChannel,
			1),
	}

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := priority_channels.CombineByFrequencyRatio[string](ctxWithCancel, channelsWithFreqRatio)
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

func TestProcessMessagesByFreqRatioAmongHighestFirstChannelGroups_InnerPriorityChannelContextCanceled(t *testing.T) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()

	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctxWithCancel, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithPriority(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio[string](
			"Free User",
			freeUserPriorityChannel,
			1),
	}

	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(ctx)
	if status != priority_channels.ReceiveInnerPriorityChannelClosed {
		t.Errorf("Expected status ReceiveInnerPriorityChannelClosed (%d), but got %d", priority_channels.ReceiveInnerPriorityChannelClosed, status)
	}
	if channelName != "Paying Customer" {
		t.Errorf("Expected channel name 'Paying Customer', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}
