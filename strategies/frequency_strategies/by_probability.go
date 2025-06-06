package frequency_strategies

import (
	"errors"
	"math/rand/v2"
	"sort"

	"github.com/dmgrit/priority-channels/strategies"
)

var (
	ErrProbabilityIsInvalid      = errors.New("probability must be between 0 and 1 (exclusive)")
	ErrProbabilitiesMustSumToOne = errors.New("sum of probabilities must be 1")
)

type ByProbability struct {
	pendingProbabilities          []probabilitySelection
	pendingProbabilitiesInitState []probabilitySelection
	currSelectedIndexes           []int
	origProbabilities             []float64
	disabledCases                 map[int]float64
	initialized                   bool
}

type probabilitySelection struct {
	Probability   float64
	AdjustedValue float64
	OriginalIndex int
}

func NewByProbability() *ByProbability {
	return &ByProbability{
		disabledCases: make(map[int]float64),
	}
}

func (s *ByProbability) Initialize(probabilities []float64) error {
	s.pendingProbabilities = make([]probabilitySelection, 0, len(probabilities))
	s.origProbabilities = make([]float64, 0, len(probabilities))

	probSum := 0.0
	for i, p := range probabilities {
		if p <= 0 || p >= 1 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          ErrProbabilityIsInvalid,
			}
		}
		s.origProbabilities = append(s.origProbabilities, p)
		probSum += p
		s.pendingProbabilities = append(s.pendingProbabilities, probabilitySelection{
			Probability:   p,
			OriginalIndex: i,
		})
	}
	if probSum != 1.0 {
		return ErrProbabilitiesMustSumToOne
	}

	sort.Slice(s.pendingProbabilities, func(i, j int) bool {
		return s.pendingProbabilities[i].Probability > s.pendingProbabilities[j].Probability ||
			(s.pendingProbabilities[i].Probability == s.pendingProbabilities[j].Probability &&
				s.pendingProbabilities[i].OriginalIndex > s.pendingProbabilities[j].OriginalIndex)
	})
	currSum := 0.0
	for i, p := range s.pendingProbabilities {
		currSum += p.Probability
		s.pendingProbabilities[i].AdjustedValue = currSum
	}

	// Save init state to be able to reset after each update on selection
	s.pendingProbabilitiesInitState = make([]probabilitySelection, 0, len(s.pendingProbabilities))
	for _, p := range s.pendingProbabilities {
		s.pendingProbabilitiesInitState = append(s.pendingProbabilitiesInitState, p)
	}
	s.initialized = true
	return nil
}

func (s *ByProbability) InitializeWithTypeAssertion(probabilities []interface{}) error {
	probabilitiesFloat64, err := strategies.ConvertWeightsWithTypeAssertion[float64]("probability", probabilities)
	if err != nil {
		return err
	}
	return s.Initialize(probabilitiesFloat64)
}

func (s *ByProbability) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	if len(s.currSelectedIndexes) < upto && len(s.pendingProbabilities) > 0 {
		remainingFlops := upto - len(s.currSelectedIndexes)
		for i := 1; i <= remainingFlops && len(s.pendingProbabilities) > 0; i++ {
			s.nextFlop()
		}
	}
	res := make([]strategies.RankedIndex, 0, upto)
	for i := 0; i < upto && i < len(s.currSelectedIndexes); i++ {
		res = append(res, strategies.RankedIndex{Index: s.currSelectedIndexes[i], Rank: i + 1})
	}
	return res, len(s.pendingProbabilities) == 0 && len(res) == len(s.currSelectedIndexes)
}

func (s *ByProbability) nextFlop() {
	i := 0
	if len(s.pendingProbabilities) > 1 {
		nextFlop := rand.Float64()
		i = sort.Search(len(s.pendingProbabilities), func(i int) bool {
			return nextFlop <= s.pendingProbabilities[i].AdjustedValue
		})
	}
	s.currSelectedIndexes = append(s.currSelectedIndexes, s.pendingProbabilities[i].OriginalIndex)
	s.pendingProbabilities = append(s.pendingProbabilities[:i], s.pendingProbabilities[i+1:]...)
	s.readjustSortedProbabilitySelectionsList(s.pendingProbabilities)
}

func (s *ByProbability) UpdateOnCaseSelected(index int) {
	s.resetSelectionState()
}

func (s *ByProbability) DisableSelectCase(index int) {
	if _, ok := s.disabledCases[index]; ok {
		return
	}
	probability := s.origProbabilities[index]
	spIndex := sort.Search(len(s.pendingProbabilitiesInitState), func(i int) bool {
		pp := s.pendingProbabilitiesInitState[i]
		return (probability > pp.Probability) ||
			(probability == pp.Probability && index >= pp.OriginalIndex)
	})
	if spIndex == len(s.pendingProbabilitiesInitState) || s.pendingProbabilitiesInitState[spIndex].OriginalIndex != index {
		// this should never happen
		return
	}
	s.pendingProbabilitiesInitState = append(s.pendingProbabilitiesInitState[:spIndex], s.pendingProbabilitiesInitState[spIndex+1:]...)
	s.disabledCases[index] = probability
	s.readjustSortedProbabilitySelectionsList(s.pendingProbabilitiesInitState)
	s.resetSelectionState()
}

func (s *ByProbability) EnableSelectCase(index int) {
	probability, ok := s.disabledCases[index]
	if !ok {
		return
	}
	delete(s.disabledCases, index)
	s.insertToSortedProbabilitySelectionsList(&s.pendingProbabilities, probability, index)
	s.insertToSortedProbabilitySelectionsList(&s.pendingProbabilitiesInitState, probability, index)
}

func (s *ByProbability) insertToSortedProbabilitySelectionsList(plist *[]probabilitySelection, probability float64, originalIndex int) {
	list := *plist
	i := sort.Search(len(list), func(i int) bool {
		return (probability > list[i].Probability) ||
			(probability == list[i].Probability && originalIndex > list[i].OriginalIndex)
	})
	*plist = append(*plist, probabilitySelection{})
	list = *plist
	copy(list[i+1:], list[i:])
	list[i] = probabilitySelection{
		Probability:   probability,
		OriginalIndex: originalIndex,
	}
	s.readjustSortedProbabilitySelectionsList(*plist)
}

func (s *ByProbability) resetSelectionState() {
	s.currSelectedIndexes = nil
	s.pendingProbabilities = make([]probabilitySelection, 0, len(s.pendingProbabilitiesInitState))
	for _, p := range s.pendingProbabilitiesInitState {
		s.pendingProbabilities = append(s.pendingProbabilities, p)
	}
}

func (s *ByProbability) readjustSortedProbabilitySelectionsList(list []probabilitySelection) {
	if len(list) == 0 {
		return
	}
	sum := 0.0
	for _, p := range list {
		sum += p.Probability
	}
	adjustedSum := 0.0
	for i := range list {
		adjustedSum += list[i].Probability / sum
		list[i].AdjustedValue = adjustedSum
	}
	// Adjust last value to 1.0 to avoid floating point errors
	list[len(list)-1].AdjustedValue = 1.0
}

func (s *ByProbability) InitializeCopy() strategies.PrioritizationStrategy[float64] {
	if !s.initialized {
		return nil
	}
	res := NewByProbability()
	_ = res.Initialize(s.origProbabilities)
	return res
}

func (s *ByProbability) InitializeCopyAsDynamicSubStrategy() strategies.DynamicSubStrategy {
	return strategies.InitializeCopyAsDynamicSubStrategy[float64](s)
}

type ByProbabilityFromFreqRatios struct {
	origFreqRatios []int
	initialized    bool
	*ByProbability
}

func NewByProbabilityFromFreqRatios() *ByProbabilityFromFreqRatios {
	return &ByProbabilityFromFreqRatios{
		ByProbability: NewByProbability(),
	}
}

func (s *ByProbabilityFromFreqRatios) Initialize(freqRatios []int) error {
	s.origFreqRatios = make([]int, 0, len(freqRatios))
	probabilitiesFloat64 := make([]float64, 0, len(freqRatios))
	totalSum := 0.0
	for i, freqRatio := range freqRatios {
		if freqRatio <= 0 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          ErrFreqRatioMustBeGreaterThanZero,
			}
		}
		s.origFreqRatios = append(s.origFreqRatios, freqRatio)
		totalSum += float64(freqRatio)
	}
	accSum := 0.0
	for i, freqRatio := range freqRatios {
		var prob float64
		if i != len(freqRatios)-1 {
			prob = float64(freqRatio) / totalSum
			accSum = accSum + prob
		} else {
			prob = 1 - accSum
		}
		probabilitiesFloat64 = append(probabilitiesFloat64, prob)
	}
	if err := s.ByProbability.Initialize(probabilitiesFloat64); err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *ByProbabilityFromFreqRatios) InitializeWithTypeAssertion(freqRatios []interface{}) error {
	freqRatiosInt, err := strategies.ConvertWeightsWithTypeAssertion[int]("frequency ratio", freqRatios)
	if err != nil {
		return err
	}
	return s.Initialize(freqRatiosInt)
}

func (s *ByProbabilityFromFreqRatios) InitializeCopy() strategies.PrioritizationStrategy[int] {
	if !s.initialized {
		return nil
	}
	res := NewByProbabilityFromFreqRatios()
	_ = res.Initialize(s.origFreqRatios)
	return res
}

func (s *ByProbabilityFromFreqRatios) InitializeCopyAsDynamicSubStrategy() strategies.DynamicSubStrategy {
	return strategies.InitializeCopyAsDynamicSubStrategy[int](s)
}
