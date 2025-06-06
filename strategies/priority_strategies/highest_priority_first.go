package priority_strategies

import (
	"errors"
	"sort"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

var ErrPriorityIsNegative = errors.New("priority cannot be negative")

type HighestAlwaysFirst struct {
	origPriorities             []int
	sortedPriorities           []sortedToOriginalIndex
	totalSortedPriorities      int
	disabledCases              map[int]sortedToOriginalIndex
	frequencyStrategyGenerator FrequencyStrategyGenerator
	initialized                bool
}

type sortedToOriginalIndex struct {
	Priority          int
	OriginalIndex     int
	SamePriorityRange *samePriorityRange
}

type samePriorityRange struct {
	FrequencyStrategy FrequencyStrategy
	indexToOrigIndex  map[int]int
	origIndexToIndex  map[int]int
	disabledCases     map[int]int
}

func newSamePriorityRange(origIndexes []int, strategy FrequencyStrategy) (*samePriorityRange, error) {
	res := &samePriorityRange{
		FrequencyStrategy: strategy,
		indexToOrigIndex:  make(map[int]int),
		origIndexToIndex:  make(map[int]int),
		disabledCases:     make(map[int]int),
	}
	weights := make([]int, 0, len(origIndexes))
	for i, origIndex := range origIndexes {
		weights = append(weights, 1)
		res.indexToOrigIndex[i] = origIndex
		res.origIndexToIndex[origIndex] = i
	}
	err := res.FrequencyStrategy.Initialize(weights)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (sp *samePriorityRange) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	rankedIndexes, allSelected := sp.FrequencyStrategy.NextSelectCasesRankedIndexes(upto)
	res := make([]strategies.RankedIndex, 0, len(rankedIndexes))
	for _, ri := range rankedIndexes {
		res = append(res, strategies.RankedIndex{
			Index: sp.indexToOrigIndex[ri.Index],
			Rank:  ri.Rank,
		})
	}
	return res, allSelected
}

func (sp *samePriorityRange) UpdateOnCaseSelected(origIndex int) {
	index, ok := sp.origIndexToIndex[origIndex]
	if !ok {
		// this should never happen
		return
	}
	sp.FrequencyStrategy.UpdateOnCaseSelected(index)
}

func (sp *samePriorityRange) DisableSelectCase(origIndex int) {
	if _, ok := sp.disabledCases[origIndex]; ok {
		return
	}
	index, ok := sp.origIndexToIndex[origIndex]
	if !ok {
		// this should never happen
		return
	}
	sp.FrequencyStrategy.DisableSelectCase(index)
	sp.disabledCases[origIndex] = index
	delete(sp.origIndexToIndex, origIndex)
	delete(sp.indexToOrigIndex, index)
}

func (sp *samePriorityRange) EnableSelectCase(origIndex int) {
	index, ok := sp.disabledCases[origIndex]
	if !ok {
		return
	}
	delete(sp.disabledCases, origIndex)
	sp.origIndexToIndex[origIndex] = index
	sp.indexToOrigIndex[index] = origIndex
	sp.FrequencyStrategy.EnableSelectCase(index)
}

func (sp *samePriorityRange) Len() int {
	return len(sp.indexToOrigIndex)
}

type FrequencyStrategy interface {
	Initialize(weights []int) error
	NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool)
	UpdateOnCaseSelected(index int)
	DisableSelectCase(index int)
	EnableSelectCase(index int)
}

func NewByHighestAlwaysFirst(options ...func(opt *HighestAlwaysFirstOptions)) *HighestAlwaysFirst {
	pcOptions := &HighestAlwaysFirstOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	var frequencyStrategyGenerator FrequencyStrategyGenerator
	if pcOptions.frequencyStrategyGenerator != nil {
		frequencyStrategyGenerator = *pcOptions.frequencyStrategyGenerator
	} else {
		frequencyStrategyGenerator = func(int) FrequencyStrategy {
			return frequency_strategies.NewByProbabilityFromFreqRatios()
		}
	}
	return &HighestAlwaysFirst{
		disabledCases:              make(map[int]sortedToOriginalIndex),
		frequencyStrategyGenerator: frequencyStrategyGenerator,
	}
}

type FrequencyStrategyGenerator func(freqRatiosNum int) FrequencyStrategy
type HighestAlwaysFirstOptions struct {
	frequencyStrategyGenerator *FrequencyStrategyGenerator
}

func WithFrequencyStrategyGenerator(frequencyStrategyGenerator FrequencyStrategyGenerator) func(opt *HighestAlwaysFirstOptions) {
	return func(opt *HighestAlwaysFirstOptions) {
		opt.frequencyStrategyGenerator = &frequencyStrategyGenerator
	}
}

func (s *HighestAlwaysFirst) Initialize(priorities []int) error {
	s.origPriorities = make([]int, 0, len(priorities))
	s.sortedPriorities = make([]sortedToOriginalIndex, 0, len(priorities))
	for i, p := range priorities {
		if p < 0 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          ErrPriorityIsNegative,
			}
		}
		s.origPriorities = append(s.origPriorities, p)
		s.sortedPriorities = append(s.sortedPriorities, sortedToOriginalIndex{
			Priority:      p,
			OriginalIndex: i,
		})
	}
	sort.Slice(s.sortedPriorities, func(i, j int) bool {
		spi := s.sortedPriorities[i]
		spj := s.sortedPriorities[j]
		return spi.Priority > spj.Priority ||
			(spi.Priority == spj.Priority && spi.OriginalIndex < spj.OriginalIndex)
	})
	s.totalSortedPriorities = len(s.sortedPriorities)
	if err := s.shrinkSamePrioritiesRanges(); err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *HighestAlwaysFirst) shrinkSamePrioritiesRanges() error {
	for i := 0; i < len(s.sortedPriorities); i++ {
		currPriority := s.sortedPriorities[i].Priority
		finishIndex := i
		for ; finishIndex < len(s.sortedPriorities)-1 &&
			s.sortedPriorities[finishIndex+1].Priority == currPriority; finishIndex++ {
		}
		if finishIndex == i {
			continue
		}
		origIndexes := make([]int, 0, finishIndex-i+1)
		for j := i; j <= finishIndex; j++ {
			origIndexes = append(origIndexes, s.sortedPriorities[j].OriginalIndex)
		}
		spr, err := newSamePriorityRange(origIndexes, s.frequencyStrategyGenerator(len(origIndexes)))
		if err != nil {
			return err
		}
		s.sortedPriorities = append(s.sortedPriorities[:i+1], s.sortedPriorities[finishIndex+1:]...)
		s.sortedPriorities[i].OriginalIndex = -1
		s.sortedPriorities[i].SamePriorityRange = spr
	}
	return nil
}

func (s *HighestAlwaysFirst) InitializeWithTypeAssertion(priorities []interface{}) error {
	prioritiesInt, err := strategies.ConvertWeightsWithTypeAssertion[int]("priority", priorities)
	if err != nil {
		return err
	}
	return s.Initialize(prioritiesInt)
}

func (s *HighestAlwaysFirst) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	res := make([]strategies.RankedIndex, 0, upto)
	nextRank := 1
	for i := 0; len(res) < upto && i < len(s.sortedPriorities); i++ {
		if s.sortedPriorities[i].SamePriorityRange != nil {
			nextIndexes, _ := s.sortedPriorities[i].SamePriorityRange.NextSelectCasesRankedIndexes(upto - len(res))
			maxRangeRank := 0
			for j := range nextIndexes {
				nextIndexes[j].Rank = nextIndexes[j].Rank + nextRank - 1
				if nextIndexes[j].Rank > maxRangeRank {
					maxRangeRank = nextIndexes[j].Rank
				}
			}
			nextRank = maxRangeRank + 1
			res = append(res, nextIndexes...)
			continue
		}
		res = append(res, strategies.RankedIndex{
			Index: s.sortedPriorities[i].OriginalIndex,
			Rank:  nextRank,
		})
		nextRank++
	}
	return res, len(res) == s.totalSortedPriorities
}

func (s *HighestAlwaysFirst) UpdateOnCaseSelected(index int) {
	sortedIndex := s.getSortedIndexByOriginalIndex(index)
	if sortedIndex == -1 {
		// this should not happen
		return
	}
	spr := s.sortedPriorities[sortedIndex].SamePriorityRange
	if spr != nil {
		spr.UpdateOnCaseSelected(index)
	}
}

func (s *HighestAlwaysFirst) DisableSelectCase(index int) {
	if index < 0 || index > len(s.origPriorities)-1 {
		return
	}
	if _, ok := s.disabledCases[index]; ok {
		return
	}
	sortedIndex := s.getSortedIndexByOriginalIndex(index)
	if sortedIndex == -1 {
		// this should not happen
		return
	}
	disabledSortedToOriginalIndex := s.sortedPriorities[sortedIndex]
	removeSortedPriority := true
	if spr := s.sortedPriorities[sortedIndex].SamePriorityRange; spr != nil {
		spr.DisableSelectCase(index)
		if spr.Len() > 0 {
			removeSortedPriority = false
		}
	}
	if removeSortedPriority {
		s.sortedPriorities = append(s.sortedPriorities[:sortedIndex], s.sortedPriorities[sortedIndex+1:]...)
	}
	s.totalSortedPriorities--
	s.disabledCases[index] = disabledSortedToOriginalIndex
}

func (s *HighestAlwaysFirst) EnableSelectCase(index int) {
	sp, ok := s.disabledCases[index]
	if !ok {
		return
	}
	delete(s.disabledCases, index)
	i := sort.Search(len(s.sortedPriorities), func(i int) bool {
		return sp.Priority >= s.sortedPriorities[i].Priority
	})
	if i == len(s.sortedPriorities) || s.sortedPriorities[i].Priority != sp.Priority {
		s.sortedPriorities = append(s.sortedPriorities, sortedToOriginalIndex{})
		copy(s.sortedPriorities[i+1:], s.sortedPriorities[i:])
		s.sortedPriorities[i] = sp
	}
	if s.sortedPriorities[i].SamePriorityRange != nil {
		s.sortedPriorities[i].SamePriorityRange.EnableSelectCase(index)
	}
	s.totalSortedPriorities++
}

func (s *HighestAlwaysFirst) getSortedIndexByOriginalIndex(index int) int {
	priority := s.origPriorities[index]
	sortedIndex, found := sort.Find(len(s.sortedPriorities), func(i int) int {
		if priority < s.sortedPriorities[i].Priority {
			return 1
		} else if priority > s.sortedPriorities[i].Priority {
			return -1
		}
		return 0
	})
	if !found {
		// this should never happen
		return -1
	}
	return sortedIndex
}

func (s *HighestAlwaysFirst) InitializeCopy() strategies.PrioritizationStrategy[int] {
	if !s.initialized {
		return nil
	}
	res := NewByHighestAlwaysFirst()
	res.frequencyStrategyGenerator = s.frequencyStrategyGenerator
	_ = res.Initialize(s.origPriorities)
	return res
}

func (s *HighestAlwaysFirst) InitializeCopyAsDynamicSubStrategy() strategies.DynamicSubStrategy {
	return strategies.InitializeCopyAsDynamicSubStrategy(s)
}
