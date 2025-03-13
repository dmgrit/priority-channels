package frequency_strategies_test

import (
	"testing"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func TestWithStrictOrderFully(t *testing.T) {
	s := frequency_strategies.NewWithStrictOrderFully()
	err := s.Initialize([]int{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("Unexpected error on initialization: %v", err)
	}

	nextIndexes, isLastIteration := s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes := []strategies.RankedIndex{
		{Index: 4, Rank: 1},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(2)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 4, Rank: 1}, {Index: 3, Rank: 2},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(3)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 4, Rank: 1}, {Index: 3, Rank: 2}, {Index: 2, Rank: 3},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 4, Rank: 1}, {Index: 3, Rank: 2}, {Index: 2, Rank: 3}, {Index: 1, Rank: 4},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 4, Rank: 1}, {Index: 3, Rank: 2}, {Index: 2, Rank: 3}, {Index: 1, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 4, Rank: 1}, {Index: 3, Rank: 2}, {Index: 2, Rank: 3}, {Index: 1, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 4, Rank: 1}, {Index: 3, Rank: 2}, {Index: 2, Rank: 3}, {Index: 1, Rank: 4},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	s.UpdateOnCaseSelected(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(4)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 3, Rank: 1}, {Index: 2, Rank: 2}, {Index: 1, Rank: 3}, {Index: 0, Rank: 4},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, false)

	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 3, Rank: 1}, {Index: 2, Rank: 2}, {Index: 1, Rank: 3}, {Index: 0, Rank: 4}, {Index: 4, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(0)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 3, Rank: 1}, {Index: 2, Rank: 2}, {Index: 1, Rank: 3}, {Index: 4, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(1)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 3, Rank: 1}, {Index: 2, Rank: 2}, {Index: 1, Rank: 3}, {Index: 4, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(1)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 3, Rank: 1}, {Index: 2, Rank: 2}, {Index: 4, Rank: 3}, {Index: 1, Rank: 4}, {Index: 0, Rank: 5},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.DisableSelectCase(4)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(5)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 3, Rank: 1}, {Index: 2, Rank: 2}, {Index: 1, Rank: 3}, {Index: 0, Rank: 4},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

}
