# priority-channels
Process Go channels by priority. 


The following use cases are supported:

### Primary use cases
- **Highest priority always first** - when we always want to process messages in order of priority
- **Processing by frequency ratio** - when we want to prevent starvation of lower priority messages

### Advanced use cases - priority channel groups
- Channel groups by highest priority first inside group and choose among groups by frequency ratio
- Channel groups by frequency ratio inside group and choose among groups by highest priority first
- Channel groups by frequency ratio inside group and choose among groups by frequency ratio
- Channel groups by highest priority first inside group and choose among groups also by highest priority first
- Graph of priority channels - any combinations of the above to multiple levels of hierarchy


## Usage

### Priority channel with highest priority always first

```go
urgentC := make(chan string) 
normalC := make(chan string) 
lowPriorityC := make(chan string)

// Wrap the Go channels in a slice of channels objects with name and priority properties
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

ch, err := priority_channels.NewByHighestAlwaysFirst(ctx, channelsWithPriority)
if err != nil {
    // handle error
}

for {
    message, channelName, ok := ch.Receive()
    if !ok {
        break
    }
    fmt.Printf("%s: %s\n", channelName, message)
}
```

### Priority channel with frequency ratio

```go
highPriorityC := make(chan string)
normalPriorityC := make(chan string)
lowPriorityC := make(chan string)

// Wrap the Go channels in a slice of channels objects with name and frequency ratio properties
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

ch, err := priority_channels.NewByFrequencyRatio(ctx, channelsWithFrequencyRatio)
if err != nil {
    // handle error
}

for {
    message, channelName, ok := ch.Receive()
    if !ok {
        break
    }
    fmt.Printf("%s: %s\n", channelName, message)
}
```

### Combination of priority channels by highest priority first

```go
urgentMessagesC := make(chan string)
payingCustomerHighPriorityC := make(chan string)
payingCustomerLowPriorityC := make(chan string)
freeUserHighPriorityC := make(chan string)
freeUserLowPriorityC := make(chan string)

urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, 
    "Urgent Messages", urgentMessagesC)
if err != nil {
    // handle error
}

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
    // handle error
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
    // handle error
}

priorityChannelsWithPriority := []priority_channel_groups.PriorityChannelWithPriority[string]{
    priority_channel_groups.NewPriorityChannelWithPriority(
        "Urgent Messages", 
        urgentMessagesPriorityChannel,
        100,
    ),
    priority_channel_groups.NewPriorityChannelWithPriority(
        "Paying Customer",
        payingCustomerPriorityChannel,
        10),
    priority_channel_groups.NewPriorityChannelWithPriority(
        "Free User",
        freeUserPriorityChannel,
        1),
}

ch, err := priority_channel_groups.CombineByHighestPriorityFirst(ctx, priorityChannelsWithPriority)
if err != nil {
    // handle error
}

for {
    message, channelName, ok := ch.Receive()
    if !ok {
        break
    }
    fmt.Printf("%s: %s\n", channelName, message)
}
```

### Combination of priority channels by frequency ratio

```go
urgentMessagesC := make(chan string)
payingCustomerHighPriorityC := make(chan string)
payingCustomerLowPriorityC := make(chan string)
freeUserHighPriorityC := make(chan string)
freeUserLowPriorityC := make(chan string)

urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, 
    "Urgent Messages", urgentMessagesC)
if err != nil {
    // handle error
}

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
    // handle error
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
    // handle error
}

priorityChannelsWithFreqRatio := []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Urgent Messages", 
        urgentMessagesPriorityChannel,
        100,
    ),
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Paying Customer",
        payingCustomerPriorityChannel,
        10),
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Free User",
        freeUserPriorityChannel,
        1),
}

ch, err := priority_channel_groups.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)
if err != nil {
    // handle error
}

for {
    message, channelName, ok := ch.Receive()
    if !ok {
        break
    }
    fmt.Printf("%s: %s\n", channelName, message)
}
```

### Complex example - combination of multiple levels of priority channels

```go
urgentMessagesC := make(chan string)
flagshipProductPayingCustomerHighPriorityC := make(chan string)
flagshipProductPayingCustomerLowPriorityC := make(chan string)
flagshipProductFreeUserHighPriorityC := make(chan string)
flagshipProductFreeUserLowPriorityC := make(chan string)
nicheProductPayingCustomerHighPriorityC := make(chan string)
nicheProductPayingCustomerLowPriorityC := make(chan string)
nicheProductFreeUserHighPriorityC := make(chan string)
nicheProductFreeUserLowPriorityC := make(chan string)


flagshipProductPayingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
    channels.NewChannelWithFreqRatio(
        "Flagship Product - Paying Customer - High Priority",
        flagshipProductPayingCustomerHighPriorityC,
        5),
    channels.NewChannelWithFreqRatio(
        "Flagship Product - Paying Customer - Low Priority",
        flagshipProductPayingCustomerLowPriorityC,
        1),
})
if err != nil {
    // handle error
}

flagshipProductFreeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
    channels.NewChannelWithFreqRatio(
        "Flagship Product - Free User - High Priority",
        flagshipProductFreeUserHighPriorityC,
        5),
    channels.NewChannelWithFreqRatio(
        "Flagship Product - Free User - Low Priority",
        flagshipProductFreeUserLowPriorityC,
        1),
})
if err != nil {
    // handle error
}

flagshipProductChannelsWithFreqRatio := []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Flagship Product - Paying Customer",
        flagshipProductPayingCustomerPriorityChannel,
        10),
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Flagship Product - Free User",
        flagshipProductFreeUserPriorityChannel,
        1),
}
flagshipProductChannelGroup, err := priority_channel_groups.CombineByFrequencyRatio(ctx, flagshipProductChannelsWithFreqRatio)
if err != nil {
    // handle error
}

nicheProductPayingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
    channels.NewChannelWithFreqRatio(
        "Niche Product - Paying Customer - High Priority",
        nicheProductPayingCustomerHighPriorityC,
        5),
    channels.NewChannelWithFreqRatio(
        "Niche Product - Paying Customer - Low Priority",
        nicheProductPayingCustomerLowPriorityC,
        1),
}) 
if err != nil {
    // handle error
}

nicheProductFreeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
    channels.NewChannelWithFreqRatio(
        "Niche Product - Free User - High Priority",
        nicheProductFreeUserHighPriorityC,
        5),
    channels.NewChannelWithFreqRatio(
        "Niche Product - Free User - Low Priority",
        nicheProductFreeUserLowPriorityC,
        1),
})
if err != nil {
    // handle error
}

nicheProductChannelsWithFreqRatio := []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Niche Product - Paying Customer", 
        nicheProductPayingCustomerPriorityChannel,
        10),
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Niche Product - Free User",
        nicheProductFreeUserPriorityChannel,
        1),
}
nicheProductChannelGroup, err := priority_channel_groups.CombineByFrequencyRatio(ctx, nicheProductChannelsWithFreqRatio)
if err != nil {
    // handle error
} 

combinedProductsPriorityChannel, err := priority_channel_groups.CombineByFrequencyRatio(ctx, []priority_channel_groups.PriorityChannelWithFreqRatio[string]{
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Flagship Product",
        flagshipProductChannelGroup,
        5),
    priority_channel_groups.NewPriorityChannelWithFreqRatio(
        "Niche Product",
        nicheProductChannelGroup,
        1),
})
if err != nil {
    // handle error
}

urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, 
    "Urgent Messages", urgentMessagesC)
if err != nil {
    // handle error
}

combinedTotalPriorityChannel, err := priority_channel_groups.CombineByHighestPriorityFirst(ctx, []priority_channel_groups.PriorityChannelWithPriority[string]{
    priority_channel_groups.NewPriorityChannelWithPriority(
        "Urgent Messages",
        urgentMessagesPriorityChannel,
        10),
    priority_channel_groups.NewPriorityChannelWithPriority(
        "Combined Products",
        combinedProductsPriorityChannel,
        1),
})
if err != nil {
    // handle error
}

for {
    message, channelName, ok := combinedTotalPriorityChannel.Receive()
    if !ok {
        break
    }
    fmt.Printf("%s: %s\n", channelName, message)
}

```

### Full working example with all use cases

```go

package main

import (
    "context"
    "fmt"
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

func main() {
    usagePattern := HighestPriorityAlwaysFirst

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
        fmt.Printf("Error: %v\n", err)
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
        payingCustomerPriorityChannel, err :=  priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
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
        
        urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx,
            "Urgent Messages", urgentMessagesC)
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
```
