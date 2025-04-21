package priority_channels

import (
	"context"
	"fmt"

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

type PriorityChannelConfig struct {
	Method                    PriorityChannelMethodConfig `json:"method"`
	Channels                  []ChannelConfig             `json:"channels"`
	AutoDisableClosedChannels bool                        `json:"autoDisableClosedChannels,omitempty"`
}

type ChannelConfig struct {
	Name                   string  `json:"name"`
	Priority               int     `json:"priority,omitempty"`
	FreqRatio              int     `json:"freqRatio,omitempty"`
	Probability            float64 `json:"probability,omitempty"`
	*PriorityChannelConfig `json:"priorityChannel,omitempty"`
}

func NewFromConfiguration[T any](ctx context.Context, config Configuration, channelNameToChannel map[string]<-chan T) (*PriorityChannel[T], error) {
	if config.PriorityChannel == nil {
		return nil, fmt.Errorf("no priority channel config found")
	}
	return newFromPriorityChannelConfig(ctx, *config.PriorityChannel, channelNameToChannel)
}

func newFromPriorityChannelConfig[T any](ctx context.Context, config PriorityChannelConfig, channelNameToChannel map[string]<-chan T) (*PriorityChannel[T], error) {
	var options []func(*PriorityChannelOptions)
	if config.AutoDisableClosedChannels {
		options = append(options, AutoDisableClosedChannels())
	}
	var isCombinedPriorityChannel bool
	for _, c := range config.Channels {
		if c.PriorityChannelConfig != nil {
			isCombinedPriorityChannel = true
			break
		}
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
				priorityChannel, err = WrapAsPriorityChannel(context.Background(), c.Name, channelNameToChannel[c.Name], options...)
			} else {
				priorityChannel, err = newFromPriorityChannelConfig[T](context.Background(), *c.PriorityChannelConfig, channelNameToChannel)
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
				priorityChannel, err = WrapAsPriorityChannel(context.Background(), c.Name, channelNameToChannel[c.Name], options...)
			} else {
				priorityChannel, err = newFromPriorityChannelConfig[T](context.Background(), *c.PriorityChannelConfig, channelNameToChannel)
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
				priorityChannel, err = WrapAsPriorityChannel(context.Background(), c.Name, channelNameToChannel[c.Name], options...)
			} else {
				priorityChannel, err = newFromPriorityChannelConfig[T](context.Background(), *c.PriorityChannelConfig, channelNameToChannel)
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
