package strategies

import (
	"errors"
	"math/rand/v2"
	"sort"
)

var (
	ErrProbabilityIsInvalid      = errors.New("probability must be between 0 and 1 (exclusive)")
	ErrProbabilitiesMustSumToOne = errors.New("sum of probabilities must be 1")
)

type ByProbability struct {
	pendingProbabilities          []probabilitySelection
	pendingProbabilitiesInitState []probabilitySelection
	currSelectedIndexes           []int
}

type probabilitySelection struct {
	Probability   float64
	AdjustedValue float64
	OriginalIndex int
}

func NewByProbability() *ByProbability {
	return &ByProbability{}
}

func (s *ByProbability) Initialize(probabilities []float64) error {
	s.pendingProbabilities = make([]probabilitySelection, 0, len(probabilities))

	probSum := 0.0
	for i, p := range probabilities {
		if p <= 0 || p >= 1 {
			return &WeightValidationError{
				ChannelIndex: i,
				Err:          ErrProbabilityIsInvalid,
			}
		}
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
		return s.pendingProbabilities[i].Probability > s.pendingProbabilities[j].Probability
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
	return nil
}

func (s *ByProbability) InitializeWithTypeAssertion(probabilities []interface{}) error {
	probabilitiesFloat64, err := convertWeightsWithTypeAssertion[float64]("probability", probabilities)
	if err != nil {
		return err
	}
	return s.Initialize(probabilitiesFloat64)
}

func (s *ByProbability) NextSelectCasesIndexes(upto int) []int {
	if len(s.pendingProbabilities) > 0 {
		i := 0
		if len(s.pendingProbabilities) > 1 {
			nextFlop := rand.Float64()
			i = sort.Search(len(s.pendingProbabilities), func(i int) bool {
				return nextFlop <= s.pendingProbabilities[i].AdjustedValue
			})
		}
		s.currSelectedIndexes = append(s.currSelectedIndexes, s.pendingProbabilities[i].OriginalIndex)
		s.pendingProbabilities = append(s.pendingProbabilities[:i], s.pendingProbabilities[i+1:]...)
		s.readjustPendingProbabilities()
	}

	res := make([]int, 0, upto)
	for i := 0; i < upto && i < len(s.currSelectedIndexes); i++ {
		res = append(res, s.currSelectedIndexes[i])
	}
	return res
}

func (s *ByProbability) UpdateOnCaseSelected(index int) {
	s.currSelectedIndexes = nil
	s.pendingProbabilities = make([]probabilitySelection, 0, len(s.pendingProbabilitiesInitState))
	for _, p := range s.pendingProbabilitiesInitState {
		s.pendingProbabilities = append(s.pendingProbabilities, p)
	}
}

func (s *ByProbability) readjustPendingProbabilities() {
	if len(s.pendingProbabilities) == 0 {
		return
	}

	sum := 0.0
	for _, p := range s.pendingProbabilities {
		sum += p.Probability
	}
	adjustedSum := 0.0
	for i := range s.pendingProbabilities {
		adjustedSum += s.pendingProbabilities[i].Probability / sum
		s.pendingProbabilities[i].AdjustedValue = adjustedSum
	}
	// Adjust last value to 1.0 to avoid floating point errors
	s.pendingProbabilities[len(s.pendingProbabilities)-1].AdjustedValue = 1.0
}
