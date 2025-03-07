package frequency_strategies

import (
	"github.com/dmgrit/priority-channels/strategies"
)

type ProbabilisticByCaseDuplication struct {
	channelName     string
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
	currIndex := 0
	for i, freqRatio := range freqRatios {
		for j := 0; j < freqRatio; j++ {
			s.selectedIndexes = append(s.selectedIndexes, strategies.RankedIndex{
				Index: i,
				Rank:  1,
			})
			currIndex++
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

}
