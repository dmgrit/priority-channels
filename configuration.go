package priority_channels

import (
	"context"
	"fmt"
	"time"

	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

type Configuration struct {
	PriorityChannel *PriorityChannelConfig `json:"priorityChannel,omitempty"`
}

type PriorityChannelMethodConfig string

const (
	ByHighestAlwaysFirstMethodConfig PriorityChannelMethodConfig = "by-highest-always-first"
	ByFrequencyRatioMethodConfig     PriorityChannelMethodConfig = "by-frequency-ratio"
	ByProbabilityMethodConfig        PriorityChannelMethodConfig = "by-probability"
)

type PriorityChannelFrequencyMethodConfig string

const (
	StrictOrderAcrossCyclesFrequencyMethodConfig          PriorityChannelFrequencyMethodConfig = "strict-order-across-cycles"
	StrictOrderFullyFrequencyMethodConfig                 PriorityChannelFrequencyMethodConfig = "strict-order-fully"
	ProbabilisticByCaseDuplicationFrequencyMethodConfig   PriorityChannelFrequencyMethodConfig = "case-duplication"
	ProbabilisticByMultipleRandCallsFrequencyMethodConfig PriorityChannelFrequencyMethodConfig = "by-probability"
)

var frequencyMethodConfigToFrequencyMethod = map[PriorityChannelFrequencyMethodConfig]FrequencyMethod{
	StrictOrderAcrossCyclesFrequencyMethodConfig:          StrictOrderAcrossCycles,
	StrictOrderFullyFrequencyMethodConfig:                 StrictOrderFully,
	ProbabilisticByCaseDuplicationFrequencyMethodConfig:   ProbabilisticByCaseDuplication,
	ProbabilisticByMultipleRandCallsFrequencyMethodConfig: ProbabilisticByMultipleRandCalls,
}

type PriorityChannelFrequencyModeConfig string

const (
	StrictOrderModeFrequencyModeConfig   PriorityChannelFrequencyModeConfig = "strict-order"
	ProbabilisticModeFrequencyModeConfig PriorityChannelFrequencyModeConfig = "probabilistic"
)

var frequencyModeConfigToFrequencyMode = map[PriorityChannelFrequencyModeConfig]FrequencyMode{
	StrictOrderModeFrequencyModeConfig:   StrictOrderMode,
	ProbabilisticModeFrequencyModeConfig: ProbabilisticMode,
}

type ChannelWaitIntervalUnitConfig string

const (
	MicrosecondsChannelWaitIntervalUnitConfig ChannelWaitIntervalUnitConfig = "microseconds"
	MillisecondsChannelWaitIntervalUnitConfig ChannelWaitIntervalUnitConfig = "milliseconds"
)

var channelWaitIntervalUnitToTimeDuration = map[ChannelWaitIntervalUnitConfig]time.Duration{
	MicrosecondsChannelWaitIntervalUnitConfig: time.Microsecond,
	MillisecondsChannelWaitIntervalUnitConfig: time.Millisecond,
}

type PriorityChannelConfig struct {
	Method                    PriorityChannelMethodConfig          `json:"method"`
	Channels                  []ChannelConfig                      `json:"channels"`
	AutoDisableClosedChannels bool                                 `json:"autoDisableClosedChannels,omitempty"`
	FrequencyMode             PriorityChannelFrequencyModeConfig   `json:"frequencyMode,omitempty"`
	FrequencyMethod           PriorityChannelFrequencyMethodConfig `json:"frequencyMethod,omitempty"`
	ChannelWaitInterval       *ChannelWaitIntervalConfig           `json:"channelWaitInterval,omitempty"`
}

type ChannelWaitIntervalConfig struct {
	Unit  ChannelWaitIntervalUnitConfig `json:"unit"`
	Value int                           `json:"value"`
}

type ChannelConfig struct {
	Name                   string  `json:"name"`
	Priority               int     `json:"priority,omitempty"`
	FreqRatio              int     `json:"freqRatio,omitempty"`
	Probability            float64 `json:"probability,omitempty"`
	*PriorityChannelConfig `json:"priorityChannel,omitempty"`
}

func NewFromConfiguration[T any](ctx context.Context,
	config Configuration,
	channelNameToChannel map[string]<-chan T,
	innerPriorityChannelsContexts map[string]context.Context) (*PriorityChannel[T], error) {
	if config.PriorityChannel == nil {
		return nil, fmt.Errorf("no priority channel config found")
	}
	if _, ok := innerPriorityChannelsContexts[""]; ok {
		return nil, fmt.Errorf("empty channel name cannot be used as a key for inner priority channels contexts")
	}
	innerPriorityChannelsContextsCopy := make(map[string]context.Context, len(innerPriorityChannelsContexts)+1)
	innerPriorityChannelsContextsCopy[""] = ctx
	for k, v := range innerPriorityChannelsContexts {
		innerPriorityChannelsContextsCopy[k] = v
	}
	return newFromPriorityChannelConfig("", *config.PriorityChannel, channelNameToChannel, innerPriorityChannelsContextsCopy)
}

func newFromPriorityChannelConfig[T any](
	channelName string,
	config PriorityChannelConfig,
	channelNameToChannel map[string]<-chan T,
	innerPriorityChannelsContexts map[string]context.Context) (*PriorityChannel[T], error) {
	var options []func(*PriorityChannelOptions)
	options = append(options, combineWithoutClone())
	if config.AutoDisableClosedChannels {
		options = append(options, AutoDisableClosedChannels())
	}
	if config.FrequencyMode != "" {
		frequencyMode, ok := frequencyModeConfigToFrequencyMode[config.FrequencyMode]
		if !ok {
			return nil, fmt.Errorf("unknown frequency mode %s", config.FrequencyMode)
		}
		options = append(options, WithFrequencyMode(frequencyMode))
	}
	if config.FrequencyMethod != "" {
		frequencyMethod, ok := frequencyMethodConfigToFrequencyMethod[config.FrequencyMethod]
		if !ok {
			return nil, fmt.Errorf("unknown frequency method %s", config.FrequencyMethod)
		}
		options = append(options, WithFrequencyMethod(frequencyMethod))
	}
	if config.ChannelWaitInterval != nil && config.ChannelWaitInterval.Unit != "" && config.ChannelWaitInterval.Value > 0 {
		unitDuration, ok := channelWaitIntervalUnitToTimeDuration[config.ChannelWaitInterval.Unit]
		if !ok {
			return nil, fmt.Errorf("unknown channel wait interval unit %s", config.ChannelWaitInterval.Unit)
		}
		options = append(options, ChannelWaitInterval(time.Duration(config.ChannelWaitInterval.Value)*unitDuration))
	}
	var isCombinedPriorityChannel bool
	for _, c := range config.Channels {
		if c.PriorityChannelConfig != nil {
			isCombinedPriorityChannel = true
			break
		}
	}
	ctx := context.Background()
	if innerCtx, ok := innerPriorityChannelsContexts[channelName]; ok {
		ctx = innerCtx
	}

	if !isCombinedPriorityChannel {
		if len(config.Channels) == 1 {
			c := config.Channels[0]
			channel, ok := channelNameToChannel[c.Name]
			if !ok {
				return nil, fmt.Errorf("channel %s not found", c.Name)
			}
			return WrapAsPriorityChannel(ctx, c.Name, channel, options...)
		}

		switch config.Method {
		case ByHighestAlwaysFirstMethodConfig:
			channelsWithPriority := make([]channels.ChannelWithPriority[T], 0, len(config.Channels))
			for _, c := range config.Channels {
				channel, ok := channelNameToChannel[c.Name]
				if !ok {
					return nil, fmt.Errorf("channel %s not found", c.Name)
				}
				channelsWithPriority = append(channelsWithPriority, channels.NewChannelWithPriority(c.Name, channel, c.Priority))
			}
			return NewByHighestAlwaysFirst(ctx, channelsWithPriority, options...)
		case ByFrequencyRatioMethodConfig:
			channelsWithFreqRatio := make([]channels.ChannelWithFreqRatio[T], 0, len(config.Channels))
			for _, c := range config.Channels {
				channel, ok := channelNameToChannel[c.Name]
				if !ok {
					return nil, fmt.Errorf("channel %s not found", c.Name)
				}
				channelsWithFreqRatio = append(channelsWithFreqRatio, channels.NewChannelWithFreqRatio(c.Name, channel, c.FreqRatio))
			}
			return NewByFrequencyRatio(ctx, channelsWithFreqRatio, options...)
		case ByProbabilityMethodConfig:
			channelsWithProbability := make([]channels.ChannelWithWeight[T, float64], 0, len(config.Channels))
			for _, c := range config.Channels {
				channel, ok := channelNameToChannel[c.Name]
				if !ok {
					return nil, fmt.Errorf("channel %s not found", c.Name)
				}
				channelsWithProbability = append(channelsWithProbability, channels.NewChannelWithWeight(c.Name, channel, c.Probability))
			}
			return NewByStrategy(ctx, frequency_strategies.NewByProbability(), channelsWithProbability, options...)
		default:
			return nil, fmt.Errorf("unknown type %s", config.Method)
		}
	}

	switch config.Method {
	case ByHighestAlwaysFirstMethodConfig:
		priorityChannelsWithPriority := make([]PriorityChannelWithPriority[T], 0, len(config.Channels))
		for _, c := range config.Channels {
			var priorityChannel *PriorityChannel[T]
			var err error
			if c.PriorityChannelConfig == nil {
				wrappedCtx := context.Background()
				if innerCtx, ok := innerPriorityChannelsContexts[c.Name]; ok {
					wrappedCtx = innerCtx
				}
				priorityChannel, err = WrapAsPriorityChannel(wrappedCtx, c.Name, channelNameToChannel[c.Name], options...)
			} else {
				priorityChannel, err = newFromPriorityChannelConfig[T](c.Name, *c.PriorityChannelConfig, channelNameToChannel, innerPriorityChannelsContexts)
			}
			if err != nil {
				return nil, err
			}
			priorityChannelsWithPriority = append(priorityChannelsWithPriority, NewPriorityChannelWithPriority(c.Name, priorityChannel, c.Priority))
		}
		return CombineByHighestAlwaysFirst(ctx, priorityChannelsWithPriority, options...)
	case ByFrequencyRatioMethodConfig:
		priorityChannelsWithFreqRatio := make([]PriorityChannelWithFreqRatio[T], 0, len(config.Channels))
		for _, c := range config.Channels {
			var priorityChannel *PriorityChannel[T]
			var err error
			if c.PriorityChannelConfig == nil {
				wrappedCtx := context.Background()
				if innerCtx, ok := innerPriorityChannelsContexts[c.Name]; ok {
					wrappedCtx = innerCtx
				}
				priorityChannel, err = WrapAsPriorityChannel(wrappedCtx, c.Name, channelNameToChannel[c.Name], options...)
			} else {
				priorityChannel, err = newFromPriorityChannelConfig[T](c.Name, *c.PriorityChannelConfig, channelNameToChannel, innerPriorityChannelsContexts)
			}
			if err != nil {
				return nil, err
			}
			priorityChannelsWithFreqRatio = append(priorityChannelsWithFreqRatio, NewPriorityChannelWithFreqRatio(c.Name, priorityChannel, c.FreqRatio))
		}
		return CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio, options...)
	case ByProbabilityMethodConfig:
		priorityChannelsWithProbability := make([]PriorityChannelWithWeight[T, float64], 0, len(config.Channels))
		for _, c := range config.Channels {
			var priorityChannel *PriorityChannel[T]
			var err error
			if c.PriorityChannelConfig == nil {
				wrappedCtx := context.Background()
				if innerCtx, ok := innerPriorityChannelsContexts[c.Name]; ok {
					wrappedCtx = innerCtx
				}
				priorityChannel, err = WrapAsPriorityChannel(wrappedCtx, c.Name, channelNameToChannel[c.Name], options...)
			} else {
				priorityChannel, err = newFromPriorityChannelConfig[T](c.Name, *c.PriorityChannelConfig, channelNameToChannel, innerPriorityChannelsContexts)
			}
			if err != nil {
				return nil, err
			}
			priorityChannelsWithProbability = append(priorityChannelsWithProbability, NewPriorityChannelWithWeight(c.Name, priorityChannel, c.Probability))
		}
		return CombineByStrategy(ctx, frequency_strategies.NewByProbability(), priorityChannelsWithProbability, options...)
	}

	return nil, fmt.Errorf("unknown type %s", config.Method)
}
