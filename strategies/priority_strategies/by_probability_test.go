package priority_strategies_test

import (
	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
	"testing"
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
