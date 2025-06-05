package frequency_strategies_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func TestWithStrictOrderAcrossCycles(t *testing.T) {
	testCases := []struct {
		name       string
		strategy   strategies.PrioritizationStrategy[int]
		assertFunc assertNextIndexesFunc
	}{
		{
			name:       "WithStrictOrderAcrossCycles - Doubly Linked List implementation",
			strategy:   frequency_strategies.NewWithStrictOrderAcrossCycles(),
			assertFunc: assertNextIndexesAndLastIteration,
		},
		{
			name:       "WithStrictOrderAcrossCycles - Array implementation",
			strategy:   frequency_strategies.NewWithStrictOrderAcrossCycles2(),
			assertFunc: assertNextIndexesAndLastIterationUnorderedForSameRank,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testWithStrictOrderAcrossCycles(t, testCase.strategy, testCase.assertFunc)
		})
	}
}

func testWithStrictOrderAcrossCycles(t *testing.T, s strategies.PrioritizationStrategy[int], assert assertNextIndexesFunc) {
	err := s.Initialize([]int{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("Unexpected error on initialization: %v", err)
	}

	// first iteration - all cases are in the zero level with zero value - all selected
	nextIndexes, isLastIteration := s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes := []strategies.RankedIndex{
		{Index: 0, Rank: 1}, {Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 3, Rank: 1}, {Index: 4, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 0, Rank: 1}, {Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 3, Rank: 1}, {Index: 4, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	// case with frequency 1 is selected - so it is moved to next level
	s.UpdateOnCaseSelected(0)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 3, Rank: 1}, {Index: 4, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 3, Rank: 1}, {Index: 4, Rank: 1},
		{Index: 0, Rank: 2},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.DisableSelectCase(3)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 4, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 4, Rank: 1},
		{Index: 0, Rank: 2},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1}, {Index: 4, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1},
		{Index: 0, Rank: 2}, {Index: 4, Rank: 2},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(0)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1},
		{Index: 4, Rank: 2},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1},
		{Index: 4, Rank: 2},
		{Index: 0, Rank: 3},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(1)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 1, Rank: 1}, {Index: 2, Rank: 1},
		{Index: 4, Rank: 2},
		{Index: 0, Rank: 3},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(1)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
		{Index: 4, Rank: 2},
		{Index: 0, Rank: 3}, {Index: 1, Rank: 3},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(2)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
		{Index: 4, Rank: 2},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
		{Index: 4, Rank: 2},
		{Index: 0, Rank: 3}, {Index: 1, Rank: 3},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
		{Index: 0, Rank: 2}, {Index: 1, Rank: 2}, {Index: 4, Rank: 2},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(2)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.UpdateOnCaseSelected(2)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 2, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.UpdateOnCaseSelected(2)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 0, Rank: 1}, {Index: 1, Rank: 1}, {Index: 4, Rank: 1}, {Index: 2, Rank: 1},
	}
	assert(t, nextIndexes, isLastIteration, expectedNextIndexes, true)
}

type assertNextIndexesFunc func(t *testing.T,
	nextIndexes []strategies.RankedIndex,
	lastIteration bool,
	expectedNextIndexes []strategies.RankedIndex,
	expectedLastIteration bool)

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

func assertNextIndexesAndLastIterationUnorderedForSameRank(t *testing.T,
	nextIndexes []strategies.RankedIndex,
	lastIteration bool,
	expectedNextIndexes []strategies.RankedIndex,
	expectedLastIteration bool) {
	sort.Slice(nextIndexes, func(i, j int) bool {
		return (nextIndexes[i].Rank == nextIndexes[j].Rank && nextIndexes[i].Index < nextIndexes[j].Index) ||
			nextIndexes[i].Rank < nextIndexes[j].Rank
	})
	sort.Slice(expectedNextIndexes, func(i, j int) bool {
		return (expectedNextIndexes[i].Rank == expectedNextIndexes[j].Rank && expectedNextIndexes[i].Index < expectedNextIndexes[j].Index) ||
			expectedNextIndexes[i].Rank < expectedNextIndexes[j].Rank
	})
	if !reflect.DeepEqual(nextIndexes, expectedNextIndexes) {
		t.Fatalf("Expected to get indexes %v, but got %v", expectedNextIndexes, nextIndexes)
	}
	if lastIteration && !expectedLastIteration {
		t.Fatalf("Expected not to be the last iteration")
	} else if !lastIteration && expectedLastIteration {
		t.Fatalf("Expected to be the last iteration")
	}
}
