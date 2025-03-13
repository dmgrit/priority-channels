package frequency_strategies_test

import (
	"testing"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func TestProbabilisticByCaseDuplication(t *testing.T) {
	s := frequency_strategies.NewProbabilisticByCaseDuplication()
	err := s.Initialize([]int{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatalf("Unexpected error on initialization: %v", err)
	}

	nextIndexes, isLastIteration := s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes := []strategies.RankedIndex{
		{Index: 0, Rank: 1},
		{Index: 1, Rank: 1}, {Index: 1, Rank: 1},
		{Index: 2, Rank: 1}, {Index: 2, Rank: 1}, {Index: 2, Rank: 1},
		{Index: 3, Rank: 1}, {Index: 3, Rank: 1}, {Index: 3, Rank: 1}, {Index: 3, Rank: 1},
		{Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.UpdateOnCaseSelected(0)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 0, Rank: 1},
		{Index: 1, Rank: 1}, {Index: 1, Rank: 1},
		{Index: 2, Rank: 1}, {Index: 2, Rank: 1}, {Index: 2, Rank: 1},
		{Index: 3, Rank: 1}, {Index: 3, Rank: 1}, {Index: 3, Rank: 1}, {Index: 3, Rank: 1},
		{Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)

	s.DisableSelectCase(2)
	nextIndexes, isLastIteration = s.NextSelectCasesRankedIndexes(1)
	expectedNextIndexes = []strategies.RankedIndex{
		{Index: 0, Rank: 1},
		{Index: 1, Rank: 1}, {Index: 1, Rank: 1},
		{Index: 3, Rank: 1}, {Index: 3, Rank: 1}, {Index: 3, Rank: 1}, {Index: 3, Rank: 1},
		{Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1}, {Index: 4, Rank: 1},
	}
	assertNextIndexesAndLastIteration(t, nextIndexes, isLastIteration, expectedNextIndexes, true)
}
