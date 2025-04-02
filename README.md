# priority-channels
Process Go channels by priority 


The following use cases are supported:

### Primary use cases
- **Highest priority always first** - when we always want to process messages [in order of priority](#priority-channel-with-highest-priority-always-first)
- **Processing by frequency ratio** - when we want to prevent starvation of lower priority messages, either  
  [with goroutines](#processing-channels-by-frequency-ratio-with-goroutines) or [with priority channel](#priority-channel-with-frequency-ratio)
- **Processing by probability** - when we want to run a simulation of processing messages [with different probabilities](#priority-channel-with-probability)

### Advanced use cases - priority channel groups
- Channel groups by highest priority first inside group and choose among groups by frequency ratio
- Channel groups by frequency ratio inside group and choose among groups by highest priority first
- Channel groups by frequency ratio inside group and choose among groups by frequency ratio
- Tree of priority channels - any combinations of the above to [multiple levels of hierarchy](#combination-of-priority-channels-to-multiple-levels-of-hierarchy)

### Advanced use cases - dynamic prioritization
- Dynamic frequency ratio selection from list of [preconfigured ratios](#priority-channel-with-dynamic-frequency-ratio) 
- Dynamic prioritization strategy selection from list of [preconfigured strategies](#priority-channel-with-dynamic-prioritization-strategy)

## Usage

### Priority channel with highest priority always first

In the following scenario: 
- Messages in the high-priority channel are processed first.  
- If the high-priority channel is empty, messages from the normal-priority-1 and normal-priority-2 channels are processed 
  interchangeably since they have the same priority.  
- The low-priority channel is processed only when the high and normal-priority channels are empty.

For a full demonstration, run the [corresponding example](examples/same-priority/main.go).

```go
highPriorityC := make(chan string) 
normalPriority1C := make(chan string)
normalPriority2C := make(chan string)
lowPriorityC := make(chan string)

// Wrap the Go channels in a slice of channels objects with name and priority properties
channelsWithPriority := []channels.ChannelWithPriority[string]{
    channels.NewChannelWithPriority(
        "High Priority", 
        highPriorityC, 
        10),
    channels.NewChannelWithPriority(
        "Normal Priority 1", 
        normalPriority1C, 
        5),
    channels.NewChannelWithPriority(
        "Normal Priority 2",
        normalPriority2C,
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

### Processing channels by frequency ratio with goroutines

In the example below: 
- Messages with high, normal, and low priorities are processed at a 10:5:1 frequency ratio.  
- Each priority level has a corresponding number of goroutines, created based on this ratio, to handle message processing.  
- Processing starts asynchronously and continues until either the given context is canceled or all channels are closed.

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

fnCallback := func(message string, channelName string, status priority_channels.ReceiveStatus) {
    // do something
}

err := priority_channels.ProcessByFrequencyRatioWithGoroutines(ctx, channelsWithFrequencyRatio, fnCallback)
if err != nil {
    // handle error
}
```

### Priority channel with frequency ratio

In the example below, messages with high, normal, and low priorities are processed at a 10:5:1 frequency ratio.

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

### Priority channel with probability

In the example below, messages with high, normal, and low priorities are processed with probabilities of 0.6, 0.25, and 0.15, respectively.

```go
highPriorityC := make(chan string)
normalPriorityC := make(chan string)
lowPriorityC := make(chan string)

// Wrap the Go channels in a slice of channels objects with name and probability value properties
channelsWithProbability := []channels.ChannelWithWeight[string, float64]{
    channels.NewChannelWithWeight(
        "High Priority", 
        highPriorityC, 
        0.6),
    channels.NewChannelWithWeight(
        "Normal Priority", 
        normalPriorityC, 
        0.25),
    channels.NewChannelWithWeight(
        "Low Priority", 
        lowPriorityC, 
        0.15),
}

ch, err := priority_channels.NewByStrategy(ctx, 
    frequency_strategies.NewByProbability(), 
    channelsWithProbability)
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

In the following scenario, we have a tree of priority channels:
- Urgent messages are always processed first.
- Two groups of channels: paying customers and free users.
- Paying customers are processed 5 times for every 1 time free users are processed.
- Within each group, high priority messages are processed 3 times for every 1 time low priority messages are processed.

For a full demonstration, run the [corresponding example](examples/multi-hierarchy/main.go).

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

### Priority channel with dynamic frequency ratio

In the following scenario, we have two channels with different preconfigured frequency ratios for different time periods.

```go
customeraC := make(chan string)
customerbC := make(chan string)

channelsWithDynamicFreqRatio := []channels.ChannelWithWeight[string, map[string]int]{
    channels.NewChannelWithWeight("Customer A", customeraC,
        map[string]int{
            "Regular":              1,
            "A-Reserved":           5,
            "B-Reserved":           1,
        }),
    channels.NewChannelWithWeight("Customer B", customerbC,
        map[string]int{
            "Regular":              1,
            "A-Reserved":           1,
            "B-Reserved":           5,
        }),
}

currentStrategySelector := func() string {
    now := time.Now()
    if now.Weekday() == time.Tuesday && now.Hour() >= 9 && now.Hour() < 12 {
        return "A-Reserved"    
    } else if now.Weekday() == time.Thursday && now.Hour() >= 17 && now.Hour() < 19 {
        return "B-Reserved"
    }
    return "Regular"
}

ch, err := priority_channels.NewDynamicByPreconfiguredFrequencyRatios(ctx,
    channelsWithDynamicFreqRatio, currentStrategySelector)
if err != nil {
    // handle error
}
```

### Priority channel with dynamic prioritization strategy

In the following scenario, we have two channels with different preconfigured prioritization strategies for different time periods.

For a full demonstration, run the [corresponding example](examples/dynamic-strategy/main.go).

```go
customeraC := make(chan string)
customerbC := make(chan string)

prioritizationMethodsByName := map[string]priority_channels.PrioritizationMethod{
    "Regular":              priority_channels.ByFrequencyRatio,
    "A-Reserved":           priority_channels.ByFrequencyRatio,
    "A-Reserved-Exclusive": priority_channels.ByHighestAlwaysFirst,
    "B-Reserved":           priority_channels.ByFrequencyRatio,
    "B-Reserved-Exclusive": priority_channels.ByHighestAlwaysFirst,
}

channelsWithWeights := []channels.ChannelWithWeight[string, map[string]interface{}]{
    channels.NewChannelWithWeight("Customer A", customeraC,
        map[string]interface{}{
            "Regular":              1,
            "A-Reserved":           5,
            "A-Reserved-Exclusive": 2,
            "B-Reserved":           1,
            "B-Reserved-Exclusive": 1,
        }),
    channels.NewChannelWithWeight("Customer B", customerbC,
        map[string]interface{}{
            "Regular":              1,
            "A-Reserved":           1,
            "A-Reserved-Exclusive": 1,
            "B-Reserved":           5,
            "B-Reserved-Exclusive": 2,
        }),
}

currentStrategySelector := func() string {
    now := time.Now()
    switch {
    case now.Weekday() == time.Tuesday && now.Hour() >= 9 && now.Hour() < 11:
        return "A-Reserved"
    case now.Weekday() == time.Tuesday && now.Hour() >= 11 && now.Hour() < 12:
        return "A-Reserved-Exclusive"
    case now.Weekday() == time.Thursday && now.Hour() >= 17 && now.Hour() < 18:
        return "B-Reserved"
    case now.Weekday() == time.Thursday && now.Hour() >= 18 && now.Hour() < 19:
        return "B-Reserved-Exclusive"
    default:
        return "Regular"
    }
}

ch, err := priority_channels.NewDynamicByPreconfiguredStrategies(ctx,
    prioritizationMethodsByName, channelsWithWeights, currentStrategySelector)
if err != nil {
    // handle error
}
```


