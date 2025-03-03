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
	NextSelectCasesIndexes(upto int) []int
	UpdateOnCaseSelected(index int)
	DisableSelectCase(index int)
}

type Dynamic struct {
	strategiesByName          map[string]DynamicSubStrategy
	currentStrategyName       string
	currentStrategySelector   func() string
	origWeightsByStrategyName map[string][]interface{}
}

func NewDynamic(
	strategiesByName map[string]DynamicSubStrategy,
	currentStrategySelector func() string) *Dynamic {
	return &Dynamic{
		strategiesByName:        strategiesByName,
		currentStrategySelector: currentStrategySelector,
	}
}

func (s *Dynamic) Initialize(weights []map[string]interface{}) error {
	s.origWeightsByStrategyName = make(map[string][]interface{})

	for channelIndex, weightByStrategyName := range weights {
		if err := s.validateChannelWeightsStrategies(channelIndex, weightByStrategyName); err != nil {
			return err
		}
		for strategyName, weight := range weightByStrategyName {
			s.origWeightsByStrategyName[strategyName] = append(s.origWeightsByStrategyName[strategyName], weight)
		}
	}

	for strategyName, strategy := range s.strategiesByName {
		if err := strategy.InitializeWithTypeAssertion(s.origWeightsByStrategyName[strategyName]); err != nil {
			return err
		}
	}
	s.currentStrategyName = s.currentStrategySelector()
	return nil
}

func (s *Dynamic) validateChannelWeightsStrategies(channelIndex int, weightByStrategyName map[string]interface{}) error {
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

func (s *Dynamic) NextSelectCasesIndexes(upto int) []int {
	currentStrategyName := s.currentStrategySelector()
	if currentStrategyName != s.currentStrategyName {
		s.currentStrategyName = currentStrategyName
	}
	strategy := s.strategiesByName[currentStrategyName]
	return strategy.NextSelectCasesIndexes(upto)
}

func (s *Dynamic) UpdateOnCaseSelected(index int) {
	strategy := s.strategiesByName[s.currentStrategyName]
	strategy.UpdateOnCaseSelected(index)
}

func (s *Dynamic) DisableSelectCase(index int) {
	for _, s := range s.strategiesByName {
		s.DisableSelectCase(index)
	}
}
