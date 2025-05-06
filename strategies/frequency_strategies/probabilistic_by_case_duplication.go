package frequency_strategies

import (
	"sort"

	"github.com/dmgrit/priority-channels/strategies"
)

type ProbabilisticByCaseDuplication struct {
	disabledCases   map[int]int
	selectedIndexes []strategies.RankedIndex
	origFreqRatios  []int
}

func NewProbabilisticByCaseDuplication() *ProbabilisticByCaseDuplication {
	return &ProbabilisticByCaseDuplication{
		disabledCases: make(map[int]int),
	}
}

func (s *ProbabilisticByCaseDuplication) Initialize(freqRatios []int) error {
	totalSum := 0
	s.origFreqRatios = make([]int, 0, len(freqRatios))
	for i, freqRatio := range freqRatios {
		if freqRatio <= 0 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          ErrFreqRatioMustBeGreaterThanZero,
			}
		}
		totalSum += freqRatio
		s.origFreqRatios = append(s.origFreqRatios, freqRatio)
	}
	s.selectedIndexes = make([]strategies.RankedIndex, 0, totalSum)
	for i, freqRatio := range freqRatios {
		for j := 0; j < freqRatio; j++ {
			s.selectedIndexes = append(s.selectedIndexes, strategies.RankedIndex{
				Index: i,
				Rank:  1,
			})
		}
	}
	return nil
}

func (s *ProbabilisticByCaseDuplication) InitializeWithTypeAssertion(freqRatios []interface{}) error {
	freqRatiosInt, err := strategies.ConvertWeightsWithTypeAssertion[int]("frequency ratio", freqRatios)
	if err != nil {
		return err
	}
	return s.Initialize(freqRatiosInt)
}

func (s *ProbabilisticByCaseDuplication) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	return s.selectedIndexes, true
}

func (s *ProbabilisticByCaseDuplication) UpdateOnCaseSelected(index int) {}

func (s *ProbabilisticByCaseDuplication) DisableSelectCase(index int) {
	if _, ok := s.disabledCases[index]; ok {
		return
	}
	if index < 0 || index >= len(s.origFreqRatios) {
		return
	}
	firstIndex, found := sort.Find(len(s.selectedIndexes), func(i int) int {
		if index > s.selectedIndexes[i].Index {
			return 1
		} else if index < s.selectedIndexes[i].Index {
			return -1
		}
		return 0
	})
	if !found {
		// this should never happen
		return
	}
	lastIndex := firstIndex + s.origFreqRatios[index] - 1
	s.selectedIndexes = append(s.selectedIndexes[:firstIndex], s.selectedIndexes[lastIndex+1:]...)
	s.disabledCases[index] = s.origFreqRatios[index]
}

func (s *ProbabilisticByCaseDuplication) EnableSelectCase(index int) {
	freqRatio, ok := s.disabledCases[index]
	if !ok {
		return
	}
	delete(s.disabledCases, index)
	for j := 0; j < freqRatio; j++ {
		s.selectedIndexes = append(s.selectedIndexes, strategies.RankedIndex{
			Index: index,
			Rank:  1,
		})
	}
}
