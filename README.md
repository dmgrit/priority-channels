# priority-channels
Process Go channels by priority 


The following use cases are supported:

### Primary use cases
- **Highest priority always first** - when we always want to process messages in order of priority
- **Processing by frequency ratio** - when we want to prevent starvation of lower priority messages

### Advanced use cases - priority channel groups
- Channel groups by highest priority first inside group and choose among groups by frequency ratio
- Channel groups by frequency ratio inside group and choose among groups by highest priority first
- Channel groups by frequency ratio inside group and choose among groups by frequency ratio
- Channel groups by highest priority first inside group and choose among groups also by highest priority first
- Tree of priority channels - any combinations of the above to multiple levels of hierarchy


## Usage

### Priority channel with highest priority always first

In the example below, messages in the high-priority channel are processed first.  
If the high-priority channel is empty, messages from the normal-priority channel are processed.  
The low-priority channel is processed only when both the high and normal-priority channels are empty.    

```go
highPriorityC := make(chan string) 
normalPriorityC := make(chan string) 
lowPriorityC := make(chan string)

// Wrap the Go channels in a slice of channels objects with name and priority properties
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

In the example below, messages with high, normal, and low priorities will be processed in a ratio of 10:5:1 respectively.

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

### Combination of priority channels to multiple levels of hierarchy

In this example, we have a tree of priority channels:
- Urgent messages are always processed first.
- Two groups of channels: paying customers and free users.
- Paying customers are processed 5 times for every 1 time free users are processed.
- Within each group, high priority messages are processed 3 times for every 1 time low priority messages are processed.

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
        3),
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
        3),
    channels.NewChannelWithFreqRatio(
        "Free User - Low Priority",
        freeUserLowPriorityC,
        1),
})
if err != nil {
    // handle error
}

combinedUsersPriorityChannel, err := priority_channels.CombineByFrequencyRatio(ctx, []priority_channels.PriorityChannelWithFreqRatio[string]{
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Paying Customer",
        payingCustomerPriorityChannel,
        5),
    priority_channels.NewPriorityChannelWithFreqRatio(
        "Free User",
        freeUserPriorityChannel,
        1),
})
if err != nil {
    // handle error
}

ch, err := priority_channels.CombineByHighestAlwaysFirst(ctx, []priority_channels.PriorityChannelWithPriority[string]{
    priority_channels.NewPriorityChannelWithPriority(
        "Urgent Messages",
        urgentMessagesPriorityChannel,
        10),
    priority_channels.NewPriorityChannelWithPriority(
        "Combined Users",
        combinedUsersPriorityChannel,
        1),
})
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


