package strategies

type ByFreqRatioWithCaseDuplication struct {
	channelName     string
	disabledCases   map[int]int
	selectedIndexes []RankedIndex
	origFreqRatios  []int
}

func NewByFreqRatioWithCasesDuplication() *ByFreqRatioWithCaseDuplication {
	return &ByFreqRatioWithCaseDuplication{
		disabledCases: make(map[int]int),
	}
}

func (s *ByFreqRatioWithCaseDuplication) Initialize(freqRatios []int) error {
	totalSum := 0
	s.origFreqRatios = make([]int, 0, len(freqRatios))
	for i, freqRatio := range freqRatios {
		if freqRatio <= 0 {
			return &WeightValidationError{
				ChannelIndex: i,
				Err:          ErrFreqRatioMustBeGreaterThanZero,
			}
		}
		totalSum += freqRatio
		s.origFreqRatios = append(s.origFreqRatios, freqRatio)
	}
	s.selectedIndexes = make([]RankedIndex, 0, totalSum)
	currIndex := 0
	for i, freqRatio := range freqRatios {
		for j := 0; j < freqRatio; j++ {
			s.selectedIndexes = append(s.selectedIndexes, RankedIndex{
				Index: i,
				Rank:  1,
			})
			currIndex++
		}
	}
	return nil
}

func (s *ByFreqRatioWithCaseDuplication) InitializeWithTypeAssertion(freqRatios []interface{}) error {
	freqRatiosInt, err := convertWeightsWithTypeAssertion[int]("frequency ratio", freqRatios)
	if err != nil {
		return err
	}
	return s.Initialize(freqRatiosInt)
}

func (s *ByFreqRatioWithCaseDuplication) NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool) {
	return s.selectedIndexes, true
}

func (s *ByFreqRatioWithCaseDuplication) UpdateOnCaseSelected(index int) {}

func (s *ByFreqRatioWithCaseDuplication) DisableSelectCase(index int) {

}
