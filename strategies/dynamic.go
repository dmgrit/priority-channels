package strategies

import (
	"fmt"
)

type InvalidNumberOfStrategiesError struct {
	NumOfStrategies         int
	ExpectedNumOfStrategies int
}

func (e *InvalidNumberOfStrategiesError) Error() string {
	return fmt.Sprintf("invalid number of strategies: %d, expected %d", e.NumOfStrategies, e.ExpectedNumOfStrategies)
}

type UnknownStrategyError struct {
	StrategyName string
}

func (e *UnknownStrategyError) Error() string {
	return fmt.Sprintf("unknown strategy %s", e.StrategyName)
}

type DynamicSubStrategy interface {
	InitializeWithTypeAssertion(weights []interface{}) error
	InitializeCopyAsDynamicSubStrategy() DynamicSubStrategy
	NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool)
	UpdateOnCaseSelected(index int)
	DisableSelectCase(index int)
	EnableSelectCase(index int)
}

func InitializeCopyAsDynamicSubStrategy[W any](s PrioritizationStrategy[W]) DynamicSubStrategy {
	strategyCopy := s.InitializeCopy()
	res, ok := strategyCopy.(DynamicSubStrategy)
	if !ok {
		return nil
	}
	return res
}

type DynamicByPreconfiguredStrategies struct {
	strategiesByName        map[string]DynamicSubStrategy
	currentStrategyName     string
	currentStrategySelector func() string
}

func NewDynamicByPreconfiguredStrategies(
	strategiesByName map[string]DynamicSubStrategy,
	currentStrategySelector func() string) *DynamicByPreconfiguredStrategies {
	return &DynamicByPreconfiguredStrategies{
		strategiesByName:        strategiesByName,
		currentStrategySelector: currentStrategySelector,
	}
}

func (s *DynamicByPreconfiguredStrategies) Initialize(weights []map[string]interface{}) error {
	origWeightsByStrategyName := make(map[string][]interface{})

	for channelIndex, weightByStrategyName := range weights {
		if err := s.validateChannelWeightsStrategies(channelIndex, weightByStrategyName); err != nil {
			return err
		}
		for strategyName, weight := range weightByStrategyName {
			origWeightsByStrategyName[strategyName] = append(origWeightsByStrategyName[strategyName], weight)
		}
	}

	for strategyName, strategy := range s.strategiesByName {
		if err := strategy.InitializeWithTypeAssertion(origWeightsByStrategyName[strategyName]); err != nil {
			return err
		}
	}
	s.currentStrategyName = s.currentStrategySelector()
	return nil
}

func (s *DynamicByPreconfiguredStrategies) InitializeCopy() PrioritizationStrategy[map[string]interface{}] {
	copyStrategies := make(map[string]DynamicSubStrategy, len(s.strategiesByName))
	for name, strategy := range s.strategiesByName {
		copyStrategies[name] = strategy.InitializeCopyAsDynamicSubStrategy()
	}
	res := NewDynamicByPreconfiguredStrategies(copyStrategies, s.currentStrategySelector)
	res.currentStrategyName = s.currentStrategyName
	return res
}

func (s *DynamicByPreconfiguredStrategies) validateChannelWeightsStrategies(channelIndex int, weightByStrategyName map[string]interface{}) error {
	if len(weightByStrategyName) != len(s.strategiesByName) {
		return &WeightValidationError{
			ChannelIndex: channelIndex,
			Err: &InvalidNumberOfStrategiesError{
				NumOfStrategies:         len(weightByStrategyName),
				ExpectedNumOfStrategies: len(s.strategiesByName),
			},
		}
	}
	for strategyName := range weightByStrategyName {
		if _, ok := s.strategiesByName[strategyName]; !ok {
			return &WeightValidationError{
				ChannelIndex: channelIndex,
				Err:          &UnknownStrategyError{StrategyName: strategyName},
			}
		}
	}
	return nil
}

func (s *DynamicByPreconfiguredStrategies) NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool) {
	currentStrategyName := s.currentStrategySelector()
	if currentStrategyName != s.currentStrategyName {
		s.currentStrategyName = currentStrategyName
	}
	strategy := s.strategiesByName[currentStrategyName]
	return strategy.NextSelectCasesRankedIndexes(upto)
}

func (s *DynamicByPreconfiguredStrategies) UpdateOnCaseSelected(index int) {
	strategy := s.strategiesByName[s.currentStrategyName]
	strategy.UpdateOnCaseSelected(index)
}

func (s *DynamicByPreconfiguredStrategies) DisableSelectCase(index int) {
	for _, s := range s.strategiesByName {
		s.DisableSelectCase(index)
	}
}

func (s *DynamicByPreconfiguredStrategies) EnableSelectCase(index int) {
	for _, s := range s.strategiesByName {
		s.EnableSelectCase(index)
	}
}
