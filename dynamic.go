package priority_channels

import (
	"context"
	"fmt"

	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
)

type InvalidPrioritizationMethodError struct {
	Method int
}

func (e *InvalidPrioritizationMethodError) Error() string {
	return fmt.Sprintf("prioritization method '%d' is not valid", e.Method)
}

type PrioritizationMethod int

const (
	ByHighestAlwaysFirst PrioritizationMethod = iota
	ByFrequencyRatio
	ByProbability
)

func NewDynamicByPreconfiguredFrequencyRatios[T any](ctx context.Context,
	channelsWithDynamicFreqRatio []channels.ChannelWithWeight[T, map[string]int],
	currentStrategySelector func() string,
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	dynamicPrioritizationMethods := make(map[string]PrioritizationMethod)
	if len(channelsWithDynamicFreqRatio) == 0 {
		return nil, ErrNoChannels
	}
	c := channelsWithDynamicFreqRatio[0]
	for name := range c.Weight() {
		dynamicPrioritizationMethods[name] = ByFrequencyRatio
	}
	channelsWithDynamicWeights := make([]channels.ChannelWithWeight[T, map[string]interface{}], 0, len(channelsWithDynamicFreqRatio))
	for _, c := range channelsWithDynamicFreqRatio {
		dynamicWeights := make(map[string]interface{})
		for name, freqRatio := range c.Weight() {
			dynamicWeights[name] = freqRatio
		}
		channelsWithDynamicWeights = append(channelsWithDynamicWeights, channels.NewChannelWithWeight(c.ChannelName(), c.MsgsC(), dynamicWeights))
	}
	return NewDynamicByPreconfiguredStrategies(ctx,
		dynamicPrioritizationMethods, channelsWithDynamicWeights, currentStrategySelector, options...)
}

func CombineDynamicByPreconfiguredFrequencyRatios[T any](ctx context.Context,
	channelsWithDynamicFreqRatio []PriorityChannelWithWeight[T, map[string]int],
	currentStrategySelector func() string,
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	dynamicPrioritizationMethods := make(map[string]PrioritizationMethod)
	if len(channelsWithDynamicFreqRatio) == 0 {
		return nil, ErrNoChannels
	}
	c := channelsWithDynamicFreqRatio[0]
	for name := range c.Weight() {
		dynamicPrioritizationMethods[name] = ByFrequencyRatio
	}
	channelsWithDynamicWeights := make([]PriorityChannelWithWeight[T, map[string]interface{}], 0, len(channelsWithDynamicFreqRatio))
	for _, c := range channelsWithDynamicFreqRatio {
		dynamicWeights := make(map[string]interface{})
		for name, freqRatio := range c.Weight() {
			dynamicWeights[name] = freqRatio
		}
		channelsWithDynamicWeights = append(channelsWithDynamicWeights, NewPriorityChannelWithWeight(c.Name(), c.PriorityChannel(), dynamicWeights))
	}
	return CombineDynamicByPreconfiguredStrategies(ctx,
		dynamicPrioritizationMethods, channelsWithDynamicWeights, currentStrategySelector, options...)
}

func NewDynamicByPreconfiguredStrategies[T any](ctx context.Context,
	dynamicPrioritizationMethods map[string]PrioritizationMethod,
	channelsWithDynamicWeights []channels.ChannelWithWeight[T, map[string]interface{}],
	currentStrategySelector func() string,
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	if len(channelsWithDynamicWeights) == 0 {
		return nil, ErrNoChannels
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	dynamicWeights := make([]dynamicWeight, 0, len(channelsWithDynamicWeights))
	for _, c := range channelsWithDynamicWeights {
		dynamicWeights = append(dynamicWeights, &c)
	}
	dynamicSubStrategies, err := getDynamicSubStrategies(levelNew, dynamicPrioritizationMethods, dynamicWeights, pcOptions)
	if err != nil {
		return nil, err
	}
	return NewByStrategy(ctx,
		strategies.NewDynamicByPreconfiguredStrategies(dynamicSubStrategies, currentStrategySelector),
		channelsWithDynamicWeights,
		options...)
}

func CombineDynamicByPreconfiguredStrategies[T any](ctx context.Context,
	dynamicPrioritizationMethods map[string]PrioritizationMethod,
	priorityChannelsWithDynamicWeights []PriorityChannelWithWeight[T, map[string]interface{}],
	currentStrategySelector func() string,
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	if len(priorityChannelsWithDynamicWeights) == 0 {
		return nil, ErrNoChannels
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	dynamicWeights := make([]dynamicWeight, 0, len(priorityChannelsWithDynamicWeights))
	for _, c := range priorityChannelsWithDynamicWeights {
		dynamicWeights = append(dynamicWeights, &c)
	}
	dynamicSubStrategies, err := getDynamicSubStrategies(levelCombine, dynamicPrioritizationMethods, dynamicWeights, pcOptions)
	if err != nil {
		return nil, err
	}
	return CombineByStrategy(ctx,
		strategies.NewDynamicByPreconfiguredStrategies(dynamicSubStrategies, currentStrategySelector),
		priorityChannelsWithDynamicWeights,
		options...)
}

type dynamicWeight interface {
	Weight() map[string]interface{}
}

func getDynamicSubStrategies(
	level priorityChannelLevel,
	dynamicPrioritizationMethods map[string]PrioritizationMethod,
	channelsWithDynamicWeights []dynamicWeight,
	pcOptions *PriorityChannelOptions,
) (map[string]strategies.DynamicSubStrategy, error) {
	strategiesByName := make(map[string]strategies.DynamicSubStrategy)
	for name, prioritizationMethod := range dynamicPrioritizationMethods {
		switch prioritizationMethod {
		case ByHighestAlwaysFirst:
			strategiesByName[name] = priority_strategies.NewByHighestAlwaysFirst()
		case ByFrequencyRatio:
			sumFreqRatios := 0
			for _, c := range channelsWithDynamicWeights {
				weight, ok := c.Weight()[name]
				if !ok {
					return nil, fmt.Errorf("cannot find weight for strategy '%s'", name)
				}
				freqRatio, ok := weight.(int)
				if !ok {
					return nil, fmt.Errorf("weight for strategy '%s' must be of type int", name)
				}
				sumFreqRatios += freqRatio
			}

			strategy, err := getFrequencyStrategy(level, pcOptions.frequencyMode, pcOptions.frequencyMethod, sumFreqRatios)
			if err != nil {
				return nil, err
			}
			dynamicSubStrategy, ok := strategy.(strategies.DynamicSubStrategy)
			if !ok {
				return nil, fmt.Errorf("cannot use strategy '%s' as dynamic sub-strategy", name)
			}
			strategiesByName[name] = dynamicSubStrategy
		case ByProbability:
			strategiesByName[name] = frequency_strategies.NewByProbability()
		default:
			return nil, &InvalidPrioritizationMethodError{Method: int(prioritizationMethod)}
		}
	}
	return strategiesByName, nil
}
