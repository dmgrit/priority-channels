# priority-channels
Process Go channels by priority 

This project is companion to https://github.com/dmgrit/priority-workers.  
  
The two projects differ mainly in **how** they process channels:

- `priority-workers` takes an **asynchronous** approach.
  It uses **goroutines** to process channels concurrently.  
  This is generally **faster**, but it allows messages to exist in an **intermediate state** -
  already read from an input channel, but still moving through the channel hierarchy, waiting to be processed.
  
  
- `priority-channels` (this package) focuses on **synchronous** processing.
  It preserves the atomic semantics of Go’s `select` statement by collapsing the entire channel hierarchy into a single `select`
  (either as a single call or a loop over `select` calls).  
  This approach is generally **slower** -especially when looping- but ensures that each message is either **fully processed or not processed at all**.
  No partial work happens.  
  It also allows for easier implementation of advanced use cases, such as dynamic prioritization and dynamic frequency ratio selection.  
  
  
## Use Cases

The following use cases are supported:

### Primary use cases
- **Processing by frequency ratio** - either [with goroutines](#processing-channels-by-frequency-ratio-with-goroutines) or [with priority channel](#priority-channel-with-frequency-ratio).
- **Highest priority always first** - when we always want to process messages [in order of priority](#priority-channel-with-highest-priority-always-first), 
  regardless of the risk of starvation of lower priority messages
- **Processing by probability** - A variant of frequency ratio processing, where messages are handled randomly 
[with probabilities](#priority-channel-with-probability) defined as floating-point numbers 

### Advanced use cases - priority channel groups
- Channel groups by highest priority first inside group and choose among groups by frequency ratio
- Channel groups by frequency ratio inside group and choose among groups by highest priority first
- Channel groups by frequency ratio inside group and choose among groups by frequency ratio
- Tree of priority channels - any combinations of the above to [multiple levels of hierarchy](#combination-of-priority-channels-to-multiple-levels-of-hierarchy)

### Advanced use cases - dynamic prioritization
- Dynamic frequency ratio selection from list of [preconfigured ratios](#priority-channel-with-dynamic-frequency-ratio) 
- Dynamic prioritization strategy selection from list of [preconfigured strategies](#priority-channel-with-dynamic-prioritization-strategy)
- Dynamic prioritization configuration that can be [fully reconfigured in runtime](#priority-consumer-with-dynamic-priority-channel-that-can-be-reconfigured-in-runtime) 

### Advanced use cases - selecting frequency method
- When using priority channels, the [frequency method](#frequency-methods) is selected automatically,
but it can also be explicitly set to choose specific behavior and performance characteristics
  
Initiation can be done either programmatically or [from a configuration](#combination-of-priority-channels-to-multiple-levels-of-hierarchy-from-configuration) 

## Installation

```shell
go get github.com/dmgrit/priority-channels
```

## Usage

Below are examples demonstrating how to use the library.  
For a detailed explanation of priority channels, refer to the [Priority Channel](#priority-channel) section.

### Processing channels by frequency ratio with goroutines

In the following example: 
- Messages with high, normal, and low priorities are processed at a frequency ratio of 10:5:1.  
- Each priority level has a corresponding number of goroutines, created based on this ratio, to handle message processing, 
total of 16 goroutines (10+5+1).
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

onMessageReceived := func(message string, channelName string) {
    // do something
}

onChannelClosed := func(channelName string) {
    fmt.Printf("Channel %s is closed\n", channelName)
}

onProcessingFinished := func(reason priority_channels.ExitReason) {
    if reason == priority_channels.ContextCancelled || 
        reason == priority_channels.NoOpenChannels {
        fmt.Printf("Processing has finished, reason %v\n", reason)
    } else {
        fmt.Printf("Processing has finished, unexpected reason %v\n", reason)
    }   
}

err := priority_channels.ProcessByFrequencyRatioWithGoroutines(ctx, 
    channelsWithFrequencyRatio, 
    onMessageReceived,
    onChannelClosed,
    onProcessingFinished)
if err != nil {
    // handle error
}
```

### Priority channel with frequency ratio

In the following example, messages with high, normal, and low priorities are processed at a frequency ratio of 10:5:1.

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

### Priority channel with probability

In the following example, messages with high, normal, and low priorities are processed with probabilities of 0.6, 0.25, and 0.15, respectively.

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
  
*The internal implementation preserves the atomic semantics of Go’s `select` statement by collapsing the entire channel hierarchy into a single `select` statement.  
For an implementation using goroutines, check-out the [priority-workers](https://github.com/dmgrit/priority-workers) companion project.

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

### Combination of priority channels to multiple levels of hierarchy from Configuration

This example is the same as the [previous one](#combination-of-priority-channels-to-multiple-levels-of-hierarchy),  
but this time, the channels tree is created using a JSON configuration.

```go
urgentMessagesC := make(chan string)
payingCustomerHighPriorityC := make(chan string)
payingCustomerLowPriorityC := make(chan string)
freeUserHighPriorityC := make(chan string)
freeUserLowPriorityC := make(chan string)

var priorityConfigurationJson = `
{
  "priorityChannel": {
    "method": "by-highest-always-first",
    "channels": [
      {
        "name": "Urgent Messages",
        "priority": 10
      },
      {
        "name": "Combined Users",
        "priority": 1,
        "priorityChannel": {
          "method": "by-frequency-ratio",
          "channels": [
            {
              "name": "Paying Customer",
              "freqRatio": 5,
              "priorityChannel": {
                "method": "by-frequency-ratio",
                "channels": [
                  {
                    "name": "Paying Customer - High Priority",
                    "freqRatio": 3
                  },
                  {
                    "name": "Paying Customer - Low Priority",
                    "freqRatio": 1
                  }
                ]
              }
            },
            {
              "name": "Free User",
              "freqRatio": 1,
              "priorityChannel": {
                "method": "by-frequency-ratio",
                "channels": [
                  {
                    "name": "Free User - High Priority",
                    "freqRatio": 3
                  },
                  {
                    "name": "Free User - Low Priority",
                    "freqRatio": 1
                  }
                ]
              }
            }
          ]
        }
      }
    ]
  }
}
`

channelNameToChannel := map[string]<-chan string{
    "Urgent Messages":                 urgentMessagesC,
    "Paying Customer - High Priority": payingCustomerHighPriorityC,
    "Paying Customer - Low Priority":  payingCustomerLowPriorityC,
    "Free User - High Priority":       freeUserHighPriorityC,
    "Free User - Low Priority":        freeUserLowPriorityC,
}

var priorityConfiguration priority_channels.Configuration
err := json.Unmarshal([]byte(priorityConfigurationJson), &priorityConfiguration)
if err != nil {
    // handle error
}	

ch, err := priority_channels.NewFromConfiguration[string](ctx, priorityConfiguration, channelNameToChannel)
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
            "Regular":    1,
            "A-Reserved": 5,
            "B-Reserved": 1,
        }),
    channels.NewChannelWithWeight("Customer B", customerbC,
        map[string]int{
            "Regular":    1,
            "A-Reserved": 1,
            "B-Reserved": 5,
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

### Priority Consumer with dynamic Priority Channel that can be reconfigured in runtime
```go
customeraC := make(chan string)
customerbC := make(chan string)

channelNameToChannel := map[string]<-chan string{
    "Customer A": customeraC,
    "Customer B": customerbC,
}

priorityConfig := priority_channels.Configuration{
    PriorityChannel: &priority_channels.PriorityChannelConfig{
        Method: priority_channels.ByFrequencyRatioMethodConfig,
        Channels: []priority_channels.ChannelConfig{
            {Name: "Customer A", FreqRatio: 5},
            {Name: "Customer B", FreqRatio: 1},
        },
    },
}

consumer, err := priority_channels.NewConsumer(ctx, channelNameToChannel, priorityConfig)
if err != nil {
    // handle error
}

deliveries, err := consumer.Consume()
if err != nil {
    // handle error
}

go func() {
    for d := range deliveries {
        fmt.Printf("%s: %s\n", d.ReceiveDetails.ChannelName, d.Msg)
    }
}()

priorityConfig2 := priority_channels.Configuration{
    PriorityChannel: &priority_channels.PriorityChannelConfig{
        Method: priority_channels.ByFrequencyRatioMethodConfig,
        Channels: []priority_channels.ChannelConfig{
            {Name: "Customer A", FreqRatio: 1},
            {Name: "Customer B", FreqRatio: 3},
        },
    },
}

err = consumer.UpdatePriorityConfiguration(priorityConfig2)
if err != nil {
    // handle error
}

consumer.StopGracefully()
```


## Priority Channel

A central concept of this library is the `PriorityChannel` struct, which allows to process channels with different prioritization strategies.  
The `PriorityChannel` behaves like a combination of a select statement and a Go channel.

```go
func (*PriorityChannel[T]) Receive() (msg T, channelName string, ok bool)
func (*PriorityChannel[T]) ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus)
func (*PriorityChannel[T]) ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus)
func (*PriorityChannel[T]) Close()
```

It takes the following properties from the select statement:
- It receives messages from a list of input channels
- Messages are received atomically - each `Receive` call gets exactly one message from one specific channel at a time, no more messages are read from any channel.
- Receive with default case is supported - if no messages are available, `ReceiveDefaultCase` is returned.
- Receive with context is supported - Receive call can have a context, and if the context is canceled, `ReceiveContextCancelled` is returned.
- The default behaviour, once any of the input channels is closed, is that any further `Receive` call will return immediately
  with `ReceiveChannelClosed` for that channel.

It takes the following properties from the Go channel:
- It is typed - it is used for receiving messages of a specific type
- It can be closed - either by canceling the context with which it is initialized or by explicitly calling the Close() method
- When PriorityChannel is closed, any further `Receive` call immediately returns `ReceivePriorityChannelClosed`

It expands on the select statement by adding the following properties:
- Each input channel has a name
- Each input channel has a weight that determines the priority or frequency ratio of the channel
- It can be combined with other priority channels to form a [tree of priority channels](#combination-of-priority-channels-to-multiple-levels-of-hierarchy)
- The behaviour of closed input channel can be modified by providing `AutoDisableClosedChannels()` option to the constructor
- If `AutoDisableClosedChannels()` is set, the closed input channel will be silently disabled and will not be selected for receiving messages.
  Once all input channels are closed, the `Receive` call will return `ReceiveNoOpenChannels` status.

## Combining priority channels

When combining priority channels, additional receive methods can be used to show more information about the source input channel of the message:
```go
func (*PriorityChannel[T]) ReceiveEx() (msg T, details ReceiveDetails, ok bool)
func (*PriorityChannel[T]) ReceiveWithContextEx(ctx context.Context) (msg T, details ReceiveDetails, status ReceiveStatus)
func (*PriorityChannel[T]) ReceiveWithDefaultCaseEx() (msg T, details ReceiveDetails, status ReceiveStatus)

type ReceiveDetails struct {
  ChannelName  string
  ChannelIndex int
  PathInTree   []ChannelNode
}

type ChannelNode struct {
  ChannelName  string
  ChannelIndex int
}
```

The returned `ReceiveDetails` struct contains the following properties:
- `ChannelName` - the name of the input channel from which the message was received
- `ChannelIndex` - the index of the input channel in the list of input channels in its direct parent priority channel
- `PathInTree` - the full path in the tree of priority channels, from the root priority-channel to the direct parent priority-channel
  of the input channel from which the message was received.

Those are optional, the original `Receive` methods are still available and can be used if the additional information is not needed.

## Frequency methods

There are several strategies that can be used to process channels with frequency ratio,
either by using goroutines, or by using priority channels with one of the following methods:
- **By select-case duplication** - using select statement with duplicated cases as a means of implementing selection 
  by frequency ratio
- **By probability** - using probability to process messages in a random order
- **With strict-order fully** - custom algorithm that maintains strict order of frequency ratio processing of the given channels
- **With strict-order across cycles** - custom algorithm that maintains strict order of frequency ratio processing of the given channels
  across frequency cycles, but does not enforce order of processing of messages within the same cycle  

The following table summarizes the characteristics of each method:  

| Method                      | Level                                                                                                            | Order           | Accuracy                                                                                                                      | Performance                                                                                                                                        |
|-----------------------------|------------------------------------------------------------------------------------------------------------------|-----------------|-------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| By Goroutines               | New Level Only,<br/>For Combine check-out [priority-workers](https://github.com/dmgrit/priority-workers) project | Probabilistic   | Relies on Go scheduler, but tests show it is very accurate<br/>unless message processing time is very short (less than 10 ms) | Fastest method, but requires more resources                                                                                                        |
| Select Case Duplication     | New Level Only                                                                                                   | Probabilistic   | Pretty accurate - using uniform distribution                                                                                  | Fast if number of cases is not too large, otherwise performance degrades                                                                           |
| By Probability              | New and Combine                                                                                                  | Probabilistic   | Least accurate for maintaining frequency ratio<br/> for not large number of received messages                                 | Moderately fast for all scenarios                                                                                                                  |
| Strict Order Fully          | New and Combine                                                                                                  | Strictest Order | Accurate                                                                                                                      | Fast if messages flow constantly from high-frequency channels, <br/>slower if messages arrive mostly from small subset of lower-frequency channels |
| Strict Order Across Cycles  | New Level Only                                                                                                   | Strict Order    | Accurate                                                                                                                      | Shares same characteristics with Strict Order Fully, but works faster                                                                              |

  
When using priority channels, the following frequency method selection algorithm is automatically applied (subject to change)  

| Level     | Order ("Mode") | Selected Method                                                                                                        |
|-----------|--------------------------|------------------------------------------------------------------------------------------------------------------------|
| New Level | Default                  | Select Case Duplication<br/>if resulting number of select cases is below threshold (250)<br/>Otherwise, By Probability |
| New Level | Probabilistic            | Same as in default order for New Level                                                                                 |
| New Level | StrictOrder              | Strict Order Across Cycles                                                                                             |
| Combine   | Default                  | By Probability                                                                                                         |
| Combine   | Probabilistic            | By Probability                                                                             |
| Combine   | StrictOrder              | Strict Order Fully                                                                                                     |

  
Upon initialization of the `PriorityChannel` struct (`NewByFrequencyRatio` and `CombineByFrequencyRatio`), 
optional `WithFrequencyMode()` or `WithFrequencyMethod()` parameters can be passed to influence the selection of the frequency method.

Same parameters can also be passed to the `NewByHighestAlwaysFirst` and `CombineByHighestAlwaysFirst` methods,
to influence the selection of the frequency method that is applied for subsets of channels having same priority, if such subsets exist.