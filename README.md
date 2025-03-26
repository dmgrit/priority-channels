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

priorityChannelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
    priority_channels.NewPriorityChannelWithPriority(
        "Urgent Messages", 
        urgentMessagesPriorityChannel,
        100,
    ),
    priority_channels.NewPriorityChannelWithPriority(
        "Paying Customer",
        payingCustomerPriorityChannel,
        10),
    priority_channels.NewPriorityChannelWithPriority(
        "Free User",
        freeUserPriorityChannel,
        1),
}

ch, err := priority_channels.CombineByHighestAlwaysFirst(ctx, priorityChannelsWithPriority)
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

priorityChannelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Urgent Messages", 
        urgentMessagesPriorityChannel,
        100,
    ),
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Paying Customer",
        payingCustomerPriorityChannel,
        10),
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Free User",
        freeUserPriorityChannel,
        1),
}

ch, err := priority_channels.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio)
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

flagshipProductChannelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Flagship Product - Paying Customer",
        flagshipProductPayingCustomerPriorityChannel,
        10),
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Flagship Product - Free User",
        flagshipProductFreeUserPriorityChannel,
        1),
}
flagshipProductChannelGroup, err := priority_channels.CombineByFrequencyRatio(ctx, flagshipProductChannelsWithFreqRatio)
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

nicheProductChannelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Niche Product - Paying Customer", 
        nicheProductPayingCustomerPriorityChannel,
        10),
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Niche Product - Free User",
        nicheProductFreeUserPriorityChannel,
        1),
}
nicheProductChannelGroup, err := priority_channels.CombineByFrequencyRatio(ctx, nicheProductChannelsWithFreqRatio)
if err != nil {
    // handle error
} 

combinedProductsPriorityChannel, err := priority_channels.CombineByFrequencyRatio(ctx, []priority_channels.PriorityChannelWithFreqRatio[string]{
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Flagship Product",
        flagshipProductChannelGroup,
        5),
    priority_channels.NewPriorityChannelWithFreqRatio(
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

combinedTotalPriorityChannel, err := priority_channels.CombineByHighestAlwaysFirst(ctx, []priority_channels.PriorityChannelWithPriority[string]{
    priority_channels.NewPriorityChannelWithPriority(
        "Urgent Messages",
        urgentMessagesPriorityChannel,
        10),
    priority_channels.NewPriorityChannelWithPriority(
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
