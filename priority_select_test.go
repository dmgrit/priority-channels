package priority_channels_test

import (
	"context"
	"testing"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func TestSelect(t *testing.T) {
	urgentC := make(chan string, 1)
	normalC := make(chan string, 1)
	lowPriorityC := make(chan string, 1)

	urgentC <- "Urgent message"
	normalC <- "Normal message"
	lowPriorityC <- "Low priority message"

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Low Priority Messages",
			lowPriorityC,
			1),
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentC,
			10),
		channels.NewChannelWithPriority(
			"Normal Messages",
			normalC,
			5),
	}

	ctx := context.Background()
	msg, channelName, status, err := priority_channels.Select(ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if channelName != "Urgent Messages" {
		t.Errorf("Expected channel name %s, but got %s", "Urgent Messages", channelName)
	}
	if status != priority_channels.ReceiveSuccess {
		t.Errorf("Expected ok to be true, but got false")
	}
	if msg != "Urgent message" {
		t.Errorf("Expected 'Urgent message' message, but got %s", msg)
	}

	msg, channelName, status, err = priority_channels.Select(ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if channelName != "Normal Messages" {
		t.Errorf("Expected channel name %s, but got %s", "Normal Messages", channelName)
	}
	if status != priority_channels.ReceiveSuccess {
		t.Errorf("Expected ok to be true, but got false")
	}
	if msg != "Normal message" {
		t.Errorf("Expected 'Normal message' message, but got %s", msg)
	}

	msg, channelName, status, err = priority_channels.Select(ctx, channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if channelName != "Low Priority Messages" {
		t.Errorf("Expected channel name %s, but got %s", "Low Priority Messages", channelName)
	}
	if status != priority_channels.ReceiveSuccess {
		t.Errorf("Expected ok to be true, but got false")
	}
	if msg != "Low priority message" {
		t.Errorf("Expected 'Low priority message' message, but got %s", msg)
	}
}

func TestSelectWithDefaultUseCase(t *testing.T) {
	urgentC := make(chan string, 1)
	normalC := make(chan string, 1)
	lowPriorityC := make(chan string, 1)

	channelsWithPriority := []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority(
			"Low Priority Messages",
			lowPriorityC,
			1),
		channels.NewChannelWithPriority(
			"Urgent Messages",
			urgentC,
			10),
		channels.NewChannelWithPriority(
			"Normal Messages",
			normalC,
			5),
	}

	msg, channelName, status, err := priority_channels.SelectWithDefaultCase(channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status != priority_channels.ReceiveDefaultCase {
		t.Errorf("Expected status default-select-case, but got %d", status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if msg != "" {
		t.Errorf("Expected empty message, but got %s", msg)
	}

	urgentC <- "Urgent message"
	normalC <- "Normal message"
	lowPriorityC <- "Low priority message"

	msg, channelName, status, err = priority_channels.SelectWithDefaultCase(channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if channelName != "Urgent Messages" {
		t.Errorf("Expected channel name %s, but got %s", "Urgent Messages", channelName)
	}
	if status != priority_channels.ReceiveSuccess {
		t.Errorf("Expected ok to be true, but got false")
	}
	if msg != "Urgent message" {
		t.Errorf("Expected 'Urgent message' message, but got %s", msg)
	}

	msg, channelName, status, err = priority_channels.SelectWithDefaultCase(channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if channelName != "Normal Messages" {
		t.Errorf("Expected channel name %s, but got %s", "Normal Messages", channelName)
	}
	if status != priority_channels.ReceiveSuccess {
		t.Errorf("Expected ok to be true, but got false")
	}
	if msg != "Normal message" {
		t.Errorf("Expected 'Normal message' message, but got %s", msg)
	}

	msg, channelName, status, err = priority_channels.SelectWithDefaultCase(channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if channelName != "Low Priority Messages" {
		t.Errorf("Expected channel name %s, but got %s", "Low Priority Messages", channelName)
	}
	if status != priority_channels.ReceiveSuccess {
		t.Errorf("Expected ok to be true, but got false")
	}
	if msg != "Low priority message" {
		t.Errorf("Expected 'Low priority message' message, but got %s", msg)
	}

	msg, channelName, status, err = priority_channels.SelectWithDefaultCase(channelsWithPriority)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if status != priority_channels.ReceiveDefaultCase {
		t.Errorf("Expected status default-select-case, but got %d", status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if msg != "" {
		t.Errorf("Expected empty message, but got %s", msg)
	}
}

func TestPrioritySelectValidation(t *testing.T) {
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
		t.Run(tc.Name+"- Select", func(t *testing.T) {
			ctx := context.Background()
			_, _, _, err := priority_channels.Select(ctx, tc.ChannelsWithPriorities)
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
		t.Run(tc.Name+"- SelectWithDefaultCase", func(t *testing.T) {
			_, _, _, err := priority_channels.SelectWithDefaultCase(tc.ChannelsWithPriorities)
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
