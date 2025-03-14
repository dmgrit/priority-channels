package frequency_strategies_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func TestInitialization(t *testing.T) {
	var testCases = []struct {
		Name     string
		Strategy priority_channels.PrioritizationStrategy[int]
	}{
		{
			Name:     "WithStrictOrderFully",
			Strategy: frequency_strategies.NewWithStrictOrderFully(),
		},
		{
			Name:     "WithStrictOrderAcrossCycles",
			Strategy: frequency_strategies.NewWithStrictOrderAcrossCycles(),
		},
		{
			Name:     "ProbabilisticByCaseDuplication",
			Strategy: frequency_strategies.NewProbabilisticByCaseDuplication(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testFrequencyStrategyInitialization(t, tc.Strategy)
		})
	}
}

func testFrequencyStrategyInitialization(t *testing.T, strategy priority_channels.PrioritizationStrategy[int]) {
	testCases := []struct {
		Name          string
		Weights       []int
		ExpectedError error
	}{
		{
			Name:    "Zero frequency ratio",
			Weights: []int{0, 1, 2, 3},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 0,
				Err:          frequency_strategies.ErrFreqRatioMustBeGreaterThanZero,
			},
		},
		{
			Name:    "Negative frequency ratio",
			Weights: []int{1, 2, 3, -4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 3,
				Err:          frequency_strategies.ErrFreqRatioMustBeGreaterThanZero,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := strategy.Initialize(tc.Weights)
			if err == nil {
				t.Fatalf("Expected error on initialization")
			}
			if !reflect.DeepEqual(err, tc.ExpectedError) {
				t.Fatalf("Expected error %v, but got %v", tc.ExpectedError, err)
			}
		})
	}
}

func TestInitializationWithTypeAssertion(t *testing.T) {
	var testCases = []struct {
		Name     string
		Strategy strategies.DynamicSubStrategy
	}{
		{
			Name:     "WithStrictOrderFully",
			Strategy: frequency_strategies.NewWithStrictOrderFully(),
		},
		{
			Name:     "WithStrictOrderAcrossCycles",
			Strategy: frequency_strategies.NewWithStrictOrderAcrossCycles(),
		},
		{
			Name:     "ProbabilisticByCaseDuplication",
			Strategy: frequency_strategies.NewProbabilisticByCaseDuplication(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testFrequencyStrategyInitializationWithTypeAssertion(t, tc.Strategy)
		})
	}
}

func testFrequencyStrategyInitializationWithTypeAssertion(t *testing.T, strategy strategies.DynamicSubStrategy) {
	testCases := []struct {
		Name          string
		Weights       []interface{}
		ExpectedError error
	}{
		{
			Name:    "Zero frequency ratio",
			Weights: []interface{}{0, 1, 2, 3},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 0,
				Err:          frequency_strategies.ErrFreqRatioMustBeGreaterThanZero,
			},
		},
		{
			Name:    "Negative frequency ratio",
			Weights: []interface{}{1, 2, 3, -4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 3,
				Err:          frequency_strategies.ErrFreqRatioMustBeGreaterThanZero,
			},
		},
		{
			Name:    "Invalid type",
			Weights: []interface{}{1, 2.2, 3, 4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 1,
				Err:          fmt.Errorf("frequency ratio must be of type int"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := strategy.InitializeWithTypeAssertion(tc.Weights)
			if err == nil {
				t.Fatalf("Expected error on initialization")
			}
			if !reflect.DeepEqual(err, tc.ExpectedError) {
				t.Fatalf("Expected error %v, but got %v", tc.ExpectedError, err)
			}
		})
	}
}
