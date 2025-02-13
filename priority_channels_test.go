package priority_channels_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/priority-channel-groups"
)

type UsagePattern int

const (
	HighestPriorityAlwaysFirst UsagePattern = iota
	FrequencyRatioForAll
	PayingCustomerAlwaysFirst_NoStarvationOfLowMessagesForSameUser
	NoStarvationOfFreeUser_HighPriorityMessagesAlwaysFirstForSameUser
	FrequencyRatioBetweenUsers_AndFreqRatioBetweenMessageTypesForSameUser
	PriorityForUrgentMessages_FrequencyRatioBetweenUsersAndOtherMessagesTypes
)

var usagePatternNames = map[UsagePattern]string{
	HighestPriorityAlwaysFirst: "Highest Priority Always First",
	FrequencyRatioForAll:       "Frequency Ratio For All",
	PayingCustomerAlwaysFirst_NoStarvationOfLowMessagesForSameUser:            "Paying Customer Always First, No Starvation Of Low Messages For Same User",
	NoStarvationOfFreeUser_HighPriorityMessagesAlwaysFirstForSameUser:         "No Starvation Of Free User, High Priority Messages Always First For Same User",
	FrequencyRatioBetweenUsers_AndFreqRatioBetweenMessageTypesForSameUser:     "Frequency Ratio Between Users And Frequency Ratio Between Message Types For Same User",
	PriorityForUrgentMessages_FrequencyRatioBetweenUsersAndOtherMessagesTypes: "Priority For Urgent Messages, Frequency Ratio Between Users And Other Message Types",
}

func TestAll(t *testing.T) {
	usagePatterns := []UsagePattern{
		HighestPriorityAlwaysFirst,
		FrequencyRatioForAll,
		PayingCustomerAlwaysFirst_NoStarvationOfLowMessagesForSameUser,
		NoStarvationOfFreeUser_HighPriorityMessagesAlwaysFirstForSameUser,
		FrequencyRatioBetweenUsers_AndFreqRatioBetweenMessageTypesForSameUser,
		PriorityForUrgentMessages_FrequencyRatioBetweenUsersAndOtherMessagesTypes,
	}
	for _, usagePattern := range usagePatterns {
		t.Run(usagePatternNames[usagePattern], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Recovered from panic: %v", r)
				}
			}()
			testExample(t, usagePattern)
		})
	}
}

func testExample(t *testing.T, usagePattern UsagePattern) {
	ctx, cancel := context.WithCancel(context.Background())

	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)
	urgentMessagesC := make(chan string)

	ch, err := getPriorityChannelByUsagePattern(
		ctx,
		usagePattern,
		payingCustomerHighPriorityC,
		payingCustomerLowPriorityC,
		freeUserHighPriorityC,
		freeUserLowPriorityC,
		urgentMessagesC)
	if err != nil {
		t.Errorf("Error: %v\n", err)
		cancel()
		return
	}

	// sending messages to individual channels
	go func() {
		for i := 1; i <= 20; i++ {
			payingCustomerHighPriorityC <- fmt.Sprintf("high priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			payingCustomerLowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			freeUserHighPriorityC <- fmt.Sprintf("high priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			freeUserLowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
	}()
	if usagePattern == PriorityForUrgentMessages_FrequencyRatioBetweenUsersAndOtherMessagesTypes {
		go func() {
			for i := 1; i <= 5; i++ {
				urgentMessagesC <- fmt.Sprintf("urgent message %d", i)
			}
		}()
	}

	go func() {
		time.Sleep(3 * time.Second)
		cancel()
	}()

	// receiving messages from the priority channel
	for {
		message, channelName, ok := ch.Receive()
		if !ok {
			break
		}
		fmt.Printf("%s: %s\n", channelName, message)
		time.Sleep(10 * time.Millisecond)
	}
}

func getPriorityChannelByUsagePattern(
	ctx context.Context,
	usagePattern UsagePattern,
	payingCustomerHighPriorityC chan string,
	payingCustomerLowPriorityC chan string,
	freeUserHighPriorityC chan string,
	freeUserLowPriorityC chan string,
	urgentMessagesC chan string,
) (priority_channels.PriorityChannel[string], error) {

	switch usagePattern {

	case HighestPriorityAlwaysFirst:
		channelsWithPriority := []channels.ChannelWithPriority[string]{
			channels.NewChannelWithPriority(
				"Paying Customer - High Priority",
				payingCustomerHighPriorityC,
				10),
			channels.NewChannelWithPriority(
				"Paying Customer - Low Priority",
				payingCustomerLowPriorityC,
				6),
			channels.NewChannelWithPriority(
				"Free User - High Priority",
				freeUserHighPriorityC,
				5),
			channels.NewChannelWithPriority(
				"Free User - Low Priority",
				freeUserLowPriorityC,
				1),
		}
		return priority_channels.NewByHighestAlwaysFirst(ctx, channelsWithPriority)

	case FrequencyRatioForAll:
		channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
			channels.NewChannelWithFreqRatio(
				"Paying Customer - High Priority",
				payingCustomerHighPriorityC,
				50),
			channels.NewChannelWithFreqRatio(
				"Paying Customer - Low Priority",
				payingCustomerLowPriorityC,
				10),
			channels.NewChannelWithFreqRatio(
				"Free User - High Priority",
				freeUserHighPriorityC,
				5),
			channels.NewChannelWithFreqRatio(
				"Free User - Low Priority",
				freeUserLowPriorityC,
				1),
		}
		return priority_channels.NewByFrequencyRatio(ctx, channelsWithFrequencyRatio)

	case PayingCustomerAlwaysFirst_NoStarvationOfLowMessagesForSameUser:
		payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
			return nil, fmt.Errorf("failed to create paying customer priority channel: %v", err)
		}

		freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
			return nil, fmt.Errorf("failed to create free user priority channel: %v", err)
		}

		priorityChannelsWithPriority := []priority_channel_groups.PriorityChannelWithPriority[string]{
			priority_channel_groups.NewPriorityChannelWithPriority(
				"Paying Customer",
				payingCustomerPriorityChannel,
				10),
			priority_channel_groups.NewPriorityChannelWithPriority(
				"Free User",
				freeUserPriorityChannel,
				1),
		}
		return priority_channel_groups.CombineByHighestPriorityFirst(ctx, priorityChannelsWithPriority)

	case NoStarvationOfFreeUser_HighPriorityMessagesAlwaysFirstForSameUser:
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
			return nil, fmt.Errorf("failed to create paying customer priority channel: %v", err)
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
			return nil, fmt.Errorf("failed to create free user priority channel: %v", err)
		}

		priorityChannelsWithFreqRatio := []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
			priority_channel_groups.NewPriorityChannelWithFreqRatio(
				"Paying Customer",
				payingCustomerPriorityChannel,
				10),
			priority_channel_groups.NewPriorityChannelWithFreqRatio(
				"Free User",
				freeUserPriorityChannel,
				1),
		}
		return priority_channel_groups.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)

	case FrequencyRatioBetweenUsers_AndFreqRatioBetweenMessageTypesForSameUser:
		payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
			return nil, fmt.Errorf("failed to create paying customer priority channel: %v", err)
		}

		freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
			return nil, fmt.Errorf("failed to create free user priority channel: %v", err)
		}

		priorityChannelsWithFreqRatio := []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
			priority_channel_groups.NewPriorityChannelWithFreqRatio(
				"Paying Customer",
				payingCustomerPriorityChannel,
				10),
			priority_channel_groups.NewPriorityChannelWithFreqRatio(
				"Free User",
				freeUserPriorityChannel,
				1),
		}
		return priority_channel_groups.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)

	case PriorityForUrgentMessages_FrequencyRatioBetweenUsersAndOtherMessagesTypes:
		payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
			return nil, fmt.Errorf("failed to create paying customer priority channel: %v", err)
		}

		freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
			return nil, fmt.Errorf("failed to create free user priority channel: %v", err)
		}

		priorityChannelsWithFreqRatio := []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
			priority_channel_groups.NewPriorityChannelWithFreqRatio(
				"Paying Customer",
				payingCustomerPriorityChannel,
				10),
			priority_channel_groups.NewPriorityChannelWithFreqRatio(
				"Free User",
				freeUserPriorityChannel,
				1),
		}
		combinedUsersAndMessageTypesPriorityChannel, err := priority_channel_groups.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)
		if err != nil {
			return nil, fmt.Errorf("failed to create combined users and message types priority channel: %v", err)
		}

		urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, "Urgent Messages", urgentMessagesC)
		if err != nil {
			return nil, fmt.Errorf("failed to create urgent message priority channel: %v", err)
		}

		return priority_channel_groups.CombineByHighestPriorityFirst(ctx, []priority_channel_groups.PriorityChannelWithPriority[string]{
			priority_channel_groups.NewPriorityChannelWithPriority(
				"Combined Users and Message Types",
				combinedUsersAndMessageTypesPriorityChannel,
				1),
			priority_channel_groups.NewPriorityChannelWithPriority(
				"Urgent Messages",
				urgentMessagesPriorityChannel,
				100),
		})

	default:
		return nil, fmt.Errorf("unsupported use case")
	}
}
