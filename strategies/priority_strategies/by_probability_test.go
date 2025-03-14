package priority_strategies_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
)

func TestByProbability(t *testing.T) {
	s := priority_strategies.NewByProbability()
	err := s.Initialize([]float64{0.1, 0.2, 0.3, 0.4})
	if err != nil {
		t.Fatalf("Unexpected error on initialization: %v", err)
	}

	for i := 1; i <= 4; i++ {
		nextIndexes, isLastIteration := s.NextSelectCasesRankedIndexes(i)
		expectedNextIndexesNumber := i
		expectedIsLastIteration := i == 4
		assertExpectedIndexesNumberAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexesNumber, expectedIsLastIteration)
	}

	s.UpdateOnCaseSelected(0)
	nextIndexes, isLastIteration := s.NextSelectCasesRankedIndexes(4)
	assertExpectedIndexesNumberAndLastIteration(t, nextIndexes, isLastIteration, 4, true)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	assertExpectedIndexesNumberAndLastIteration(t, nextIndexes, isLastIteration, 3, false)

	s.DisableSelectCase(0)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	assertExpectedIndexesNumberAndLastIteration(t, nextIndexes, isLastIteration, 3, true)
}

func assertExpectedIndexesNumberAndLastIteration(t *testing.T,
	nextIndexes []strategies.RankedIndex,
	lastIteration bool,
	expectedNextIndexesNum int,
	expectedLastIteration bool) {
	if len(nextIndexes) != expectedNextIndexesNum {
		t.Fatalf("Expected to get %d indexes, but got %d", expectedNextIndexesNum, len(nextIndexes))
	}
	var m = make(map[int]struct{})
	for _, index := range nextIndexes {
		if index.Index < 0 || index.Index > 3 {
			t.Fatalf("Index should be between 0 and 3, but got %d", index.Index)
		}
		if _, ok := m[index.Index]; ok {
			t.Fatalf("Index %d should not be repeated", index.Index)
		}
		m[index.Index] = struct{}{}
	}
	if lastIteration && !expectedLastIteration {
		t.Fatalf("Expected not to be the last iteration")
	} else if !lastIteration && expectedLastIteration {
		t.Fatalf("Expected to be the last iteration")
	}
}

func TestByProbability_Initialization(t *testing.T) {
	testCases := []struct {
		Name          string
		Weights       []float64
		ExpectedError error
	}{
		{
			Name:    "Zero probability",
			Weights: []float64{0.0, 0.3, 0.3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 0,
				Err:          priority_strategies.ErrProbabilityIsInvalid,
			},
		},
		{
			Name:    "Negative probability",
			Weights: []float64{0.1, -0.2, 0.3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 1,
				Err:          priority_strategies.ErrProbabilityIsInvalid,
			},
		},
		{
			Name:    "Probability above 1",
			Weights: []float64{1.1, 0.2, 0.3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 0,
				Err:          priority_strategies.ErrProbabilityIsInvalid,
			},
		},
		{
			Name:          "Probabilities sum is not 1",
			Weights:       []float64{0.1, 0.2, 0.3},
			ExpectedError: priority_strategies.ErrProbabilitiesMustSumToOne,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			strategy := priority_strategies.NewByProbability()
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

func TestByProbability_InitializationWithTypeAssertion(t *testing.T) {
	testCases := []struct {
		Name          string
		Weights       []interface{}
		ExpectedError error
	}{
		{
			Name:    "Zero probability",
			Weights: []interface{}{0.0, 0.3, 0.3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 0,
				Err:          priority_strategies.ErrProbabilityIsInvalid,
			},
		},
		{
			Name:    "Negative probability",
			Weights: []interface{}{0.1, -0.2, 0.3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 1,
				Err:          priority_strategies.ErrProbabilityIsInvalid,
			},
		},
		{
			Name:    "Probability above 1",
			Weights: []interface{}{1.1, 0.2, 0.3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 0,
				Err:          priority_strategies.ErrProbabilityIsInvalid,
			},
		},
		{
			Name:          "Probabilities sum is not 1",
			Weights:       []interface{}{0.1, 0.2, 0.3},
			ExpectedError: priority_strategies.ErrProbabilitiesMustSumToOne,
		},
		{
			Name:    "Invalid type",
			Weights: []interface{}{0.1, 0.2, 3, 0.4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 2,
				Err:          fmt.Errorf("probability must be of type float64"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			strategy := priority_strategies.NewByProbability()
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
