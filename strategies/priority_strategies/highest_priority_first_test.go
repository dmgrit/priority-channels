package priority_strategies

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func TestHighestPriorityFirst_ShrinkSamePriorityRanges(t *testing.T) {
	var testCases = []struct {
		name                     string
		priorities               []int
		expectedSortedPriorities []sortedToOriginalIndex
	}{
		{
			name:                     "empty",
			priorities:               []int{},
			expectedSortedPriorities: []sortedToOriginalIndex{},
		},
		{
			name:       "no duplicates",
			priorities: []int{1, 2, 3},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 3, OriginalIndex: 2},
				{Priority: 2, OriginalIndex: 1},
				{Priority: 1, OriginalIndex: 0}},
		},
		{
			name:       "one duplicate - beginning",
			priorities: []int{1, 1, 2, 3},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 3, OriginalIndex: 3},
				{Priority: 2, OriginalIndex: 2},
				{Priority: 1, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 0, 1: 1},
					origIndexToIndex: map[int]int{0: 0, 1: 1},
				}},
			},
		},
		{
			name:       "one duplicate - middle",
			priorities: []int{1, 2, 2, 3},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 3, OriginalIndex: 3},
				{Priority: 2, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 1, 1: 2},
					origIndexToIndex: map[int]int{1: 0, 2: 1},
				}},
				{Priority: 1, OriginalIndex: 0},
			},
		},
		{
			name:       "one duplicate - end",
			priorities: []int{1, 2, 3, 3},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 3, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 2, 1: 3},
					origIndexToIndex: map[int]int{2: 0, 3: 1},
				}},
				{Priority: 2, OriginalIndex: 1},
				{Priority: 1, OriginalIndex: 0},
			},
		},
		{
			name:       "one duplicate - priority zero should be legal",
			priorities: []int{0, 1, 0, 2},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 2, OriginalIndex: 3},
				{Priority: 1, OriginalIndex: 1},
				{Priority: 0, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 0, 1: 2},
					origIndexToIndex: map[int]int{0: 0, 2: 1},
				}},
			},
		},
		{
			name:       "multiple duplicates - sorted ascending",
			priorities: []int{1, 1, 2, 2, 2, 3, 4, 4, 4, 4, 5, 6, 6},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 6, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 11, 1: 12},
					origIndexToIndex: map[int]int{11: 0, 12: 1},
				}},
				{Priority: 5, OriginalIndex: 10},
				{Priority: 4, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 6, 1: 7, 2: 8, 3: 9},
					origIndexToIndex: map[int]int{6: 0, 7: 1, 8: 2, 9: 3},
				}},
				{Priority: 3, OriginalIndex: 5},
				{Priority: 2, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 2, 1: 3, 2: 4},
					origIndexToIndex: map[int]int{2: 0, 3: 1, 4: 2},
				}},
				{Priority: 1, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 0, 1: 1},
					origIndexToIndex: map[int]int{0: 0, 1: 1},
				}},
			},
		},
		{
			name:       "multiple duplicates - shuffled #1",
			priorities: []int{4, 6, 5, 2, 2, 4, 3, 1, 2, 6, 4, 4, 1},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 6, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 1, 1: 9},
					origIndexToIndex: map[int]int{1: 0, 9: 1},
				}},
				{Priority: 5, OriginalIndex: 2},
				{Priority: 4, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 0, 1: 5, 2: 10, 3: 11},
					origIndexToIndex: map[int]int{0: 0, 5: 1, 10: 2, 11: 3},
				}},
				{Priority: 3, OriginalIndex: 6},
				{Priority: 2, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 3, 1: 4, 2: 8},
					origIndexToIndex: map[int]int{3: 0, 4: 1, 8: 2},
				}},
				{Priority: 1, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 7, 1: 12},
					origIndexToIndex: map[int]int{7: 0, 12: 1},
				}},
			},
		},
		{
			name:       "multiple duplicates - shuffled #2",
			priorities: []int{4, 4, 3, 5, 6, 4, 6, 1, 1, 2, 4, 2, 2},
			expectedSortedPriorities: []sortedToOriginalIndex{
				{Priority: 6, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 4, 1: 6},
					origIndexToIndex: map[int]int{4: 0, 6: 1},
				}},
				{Priority: 5, OriginalIndex: 3},
				{Priority: 4, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 0, 1: 1, 2: 5, 3: 10},
					origIndexToIndex: map[int]int{0: 0, 1: 1, 5: 2, 10: 3},
				}},
				{Priority: 3, OriginalIndex: 2},
				{Priority: 2, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 9, 1: 11, 2: 12},
					origIndexToIndex: map[int]int{9: 0, 11: 1, 12: 2},
				}},
				{Priority: 1, OriginalIndex: -1, SamePriorityRange: &samePriorityRange{
					indexToOrigIndex: map[int]int{0: 7, 1: 8},
					origIndexToIndex: map[int]int{7: 0, 8: 1},
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewByHighestAlwaysFirst()
			err := s.Initialize(tc.priorities)
			if err != nil {
				t.Fatalf("Unexpected error on initializing: %v", err)
			}
			if len(s.sortedPriorities) != len(tc.expectedSortedPriorities) {
				t.Fatalf("expected sortedPriorities to have length %d, got %d", len(tc.expectedSortedPriorities), len(s.sortedPriorities))
			}
			for i, expected := range tc.expectedSortedPriorities {
				spi := s.sortedPriorities[i]
				if spi.Priority != expected.Priority {
					t.Fatalf("expected sortedPriorities[%d].Priority to be %v, got %v", i, expected.Priority, spi.Priority)
				}
				if spi.OriginalIndex != expected.OriginalIndex {
					t.Fatalf("expected sortedPriorities[%d].OriginalIndex to be %v, got %v", i, expected.OriginalIndex, spi.OriginalIndex)
				}
				if spi.SamePriorityRange == nil && expected.SamePriorityRange != nil {
					t.Fatalf("expected sortedPriorities[%d].SamePriorityRange to be not nil", i)
				} else if spi.SamePriorityRange != nil && expected.SamePriorityRange == nil {
					t.Fatalf("expected sortedPriorities[%d].SamePriorityRange to be nil", i)
				}
				if spi.SamePriorityRange == nil || expected.SamePriorityRange == nil {
					continue
				}
				if !reflect.DeepEqual(spi.SamePriorityRange.origIndexToIndex, expected.SamePriorityRange.origIndexToIndex) {
					t.Fatalf("expected sortedPriorities[%d].SamePriorityRange.origIndexToIndex to be %v, got %v", i,
						expected.SamePriorityRange.origIndexToIndex, spi.SamePriorityRange.origIndexToIndex)
				}
				if !reflect.DeepEqual(spi.SamePriorityRange.indexToOrigIndex, expected.SamePriorityRange.indexToOrigIndex) {
					t.Fatalf("expected sortedPriorities[%d].SamePriorityRange.indexToOrigIndex to be %v, got %v", i,
						expected.SamePriorityRange.indexToOrigIndex, spi.SamePriorityRange.indexToOrigIndex)
				}
			}
		})
	}
}

func TestHighestAlwaysFirst(t *testing.T) {
	s := NewByHighestAlwaysFirst(WithFrequencyStrategyGenerator(func(int) FrequencyStrategy { return frequency_strategies.NewWithStrictOrderAcrossCycles() }))
	err := s.Initialize([]int{1, 2, 3, 3, 3, 4, 5})
	if err != nil {
		t.Fatalf("Unexpected error on initialization: %v", err)
	}

	nextIndexes, isLastIteration := s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes := []strategies.RankedIndex{
		{Index: 6, Rank: 1},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(2)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 2, Rank: 3}, {Index: 3, Rank: 3}, {Index: 4, Rank: 3},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(6)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 2, Rank: 3}, {Index: 3, Rank: 3}, {Index: 4, Rank: 3},
		{Index: 1, Rank: 4},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(7)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 2, Rank: 3}, {Index: 3, Rank: 3}, {Index: 4, Rank: 3},
		{Index: 1, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(6)
	s.UpdateOnCaseSelected(0)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(7)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 2, Rank: 3}, {Index: 3, Rank: 3}, {Index: 4, Rank: 3},
		{Index: 1, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 2, Rank: 3}, {Index: 3, Rank: 3}, {Index: 4, Rank: 3},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.UpdateOnCaseSelected(2)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(7)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 3, Rank: 3}, {Index: 4, Rank: 3}, {Index: 2, Rank: 4},
		{Index: 1, Rank: 5}, {Index: 0, Rank: 6},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 3, Rank: 3}, {Index: 4, Rank: 3},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(7)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 3, Rank: 3}, {Index: 2, Rank: 4}, {Index: 4, Rank: 4},
		{Index: 1, Rank: 5}, {Index: 0, Rank: 6},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 3, Rank: 3},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.DisableSelectCase(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(6)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 6, Rank: 1}, {Index: 5, Rank: 2},
		{Index: 3, Rank: 3}, {Index: 2, Rank: 4},
		{Index: 1, Rank: 5}, {Index: 0, Rank: 6},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.DisableSelectCase(6)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 5, Rank: 1},
		{Index: 3, Rank: 2}, {Index: 2, Rank: 3},
		{Index: 1, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)
}

func assertNextIndexesAndLastIteration(t *testing.T,
	nextIndexes []strategies.RankedIndex,
	lastIteration bool,
	expectedNextIndexes []strategies.RankedIndex,
	expectedLastIteration bool) {
	if !reflect.DeepEqual(nextIndexes, expectedNextIndexes) {
		t.Fatalf("Expected to get indexes %v, but got %v", expectedNextIndexes, nextIndexes)
	}
	if lastIteration && !expectedLastIteration {
		t.Fatalf("Expected not to be the last iteration")
	} else if !lastIteration && expectedLastIteration {
		t.Fatalf("Expected to be the last iteration")
	}
}

func TestByHighestAlwaysFirst_Initialization(t *testing.T) {
	testCases := []struct {
		Name          string
		Weights       []int
		ExpectedError error
	}{
		{
			Name:    "Negative priority",
			Weights: []int{1, 2, 3, -4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 3,
				Err:          ErrPriorityIsNegative,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			strategy := NewByHighestAlwaysFirst()
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

func TestByHighestAlwaysFirst_InitializationWithTypeAssertion(t *testing.T) {
	testCases := []struct {
		Name          string
		Weights       []interface{}
		ExpectedError error
	}{
		{
			Name:    "Negative priority",
			Weights: []interface{}{1, 2, 3, -4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 3,
				Err:          ErrPriorityIsNegative,
			},
		},
		{
			Name:    "Invalid type",
			Weights: []interface{}{1, 2.2, 3, 4},
			ExpectedError: &strategies.WeightValidationError{
				ChannelIndex: 1,
				Err:          fmt.Errorf("priority must be of type int"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			strategy := NewByHighestAlwaysFirst()
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
