package priority_channels_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
)

func TestWrapAsPriorityChannelValidation(t *testing.T) {
	_, err := priority_channels.WrapAsPriorityChannel(
		context.Background(), "", make(chan string))
	if err == nil {
		t.Fatalf("Expected validation error")
	}

	expectedErrorMessage := priority_channels.ErrEmptyChannelName.Error()
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error %s, but got: %v", expectedErrorMessage, err)
	}
}

func TestWrappedClosedChannel(t *testing.T) {
	msgsC := make(chan string)
	wrappedChannel, err := priority_channels.WrapAsPriorityChannel(
		context.Background(), "test", msgsC)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	close(msgsC)

	_, channelName, status := wrappedChannel.ReceiveWithContext(context.Background())
	if status != priority_channels.ReceiveInputChannelClosed {
		t.Errorf("Expected status ReceiveChannelsClosed (%v), but got %v", priority_channels.ReceiveInputChannelClosed, status)
	}
	if channelName != "test" {
		t.Errorf("Expected channel name 'test', but got %s", channelName)
	}
}

func TestWrappedClosedChannel_WithAutoDisableOnClosedChannel(t *testing.T) {
	msgsC := make(chan string)
	wrappedChannel, err := priority_channels.WrapAsPriorityChannel(
		context.Background(), "test", msgsC, priority_channels.AutoDisableClosedChannels())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	close(msgsC)

	_, channelName, status := wrappedChannel.ReceiveWithContext(context.Background())
	if status != priority_channels.ReceiveNoReceivablePath {
		t.Errorf("Expected status ReceiveNoReceivablePath (%v), but got %v", priority_channels.ReceiveNoReceivablePath, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty priority channel name, but got %s", channelName)
	}
}

func TestWrappedChannel_AutoDisableOnClosedChannel(t *testing.T) {
	ctx := context.Background()

	msgsC := make(chan string)

	go func() {
		for i := 1; i <= 20; i++ {
			msgsC <- fmt.Sprintf("message %d", i)
		}
		close(msgsC)
	}()

	ch, err := priority_channels.WrapAsPriorityChannel(ctx, "test", msgsC, priority_channels.AutoDisableClosedChannels())
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	receivedMessagesCount := 0
	for {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != priority_channels.ReceiveSuccess {
			if receivedMessagesCount != 20 {
				t.Errorf("Expected to receive 60 messages, but got %d", receivedMessagesCount)
			}
			if status != priority_channels.ReceiveNoReceivablePath {
				t.Errorf("Expected to receive 'no receivable path' status on closure (%v), but got %v",
					priority_channels.ReceiveNoReceivablePath, status)
			}
			break
		}
		receivedMessagesCount++
		fmt.Printf("%s: %s\n", channelName, message)
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWrappedChannelClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wrappedChannel, err := priority_channels.WrapAsPriorityChannel(
		ctx, "test", make(chan string))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	cancel()
	_, channelName, status := wrappedChannel.ReceiveWithContext(context.Background())
	if status != priority_channels.ReceivePriorityChannelClosed {
		t.Errorf("Expected status ReceivePriorityChannelClosed (%v), but got %v",
			priority_channels.ReceivePriorityChannelClosed, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty priority channel name, but got %s", channelName)
	}
}

func TestCombinedWrappedChannelClosed(t *testing.T) {
	ctx := context.Background()
	ctx2, cancel := context.WithCancel(context.Background())
	wrappedChannel, err := priority_channels.WrapAsPriorityChannel(
		ctx2, "channel-1", make(chan string))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	cancel()

	otherWrappedChannel, err := priority_channels.WrapAsPriorityChannel(ctx, "channel-2", make(chan string))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst(ctx, []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("wrapped-channel-1", wrappedChannel, 5),
		priority_channels.NewPriorityChannelWithPriority("wrapped-channel-2", otherWrappedChannel, 1),
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, channelName, status := ch.ReceiveWithContext(context.Background())
	if status != priority_channels.ReceiveInnerPriorityChannelClosed {
		t.Errorf("Expected status ReceiveInnerPriorityChannelClosed (%v), but got %v",
			priority_channels.ReceiveInnerPriorityChannelClosed, status)
	}
	if channelName != "wrapped-channel-1" {
		t.Errorf("Expected empty priority channel name 'wrapped_channel-1', but got %s", channelName)
	}
}
