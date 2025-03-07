package strategies

import (
	"errors"
	"sort"
)

var ErrPriorityIsNegative = errors.New("priority cannot be negative")

type HighestAlwaysFirst struct {
	origPriorities   []int
	sortedPriorities []sortedToOriginalIndex
	disabledCases    map[int]int
}

type sortedToOriginalIndex struct {
	Priority      int
	OriginalIndex int
}

func NewByHighestAlwaysFirst() *HighestAlwaysFirst {
	return &HighestAlwaysFirst{
		disabledCases: make(map[int]int),
	}
}

func (s *HighestAlwaysFirst) Initialize(priorities []int) error {
	s.origPriorities = priorities
	s.sortedPriorities = make([]sortedToOriginalIndex, 0, len(priorities))
	for i, p := range priorities {
		if p < 0 {
			return &WeightValidationError{
				ChannelIndex: i,
				Err:          ErrPriorityIsNegative,
			}
		}
		s.sortedPriorities = append(s.sortedPriorities, sortedToOriginalIndex{
			Priority:      p,
			OriginalIndex: i,
		})
	}
	sort.Slice(s.sortedPriorities, func(i, j int) bool {
		spi := s.sortedPriorities[i]
		spj := s.sortedPriorities[j]
		return spi.Priority > spj.Priority ||
			(spi.Priority == spj.Priority && spi.OriginalIndex > spj.OriginalIndex)
	})
	return nil
}

func (s *HighestAlwaysFirst) InitializeWithTypeAssertion(priorities []interface{}) error {
	prioritiesInt, err := ConvertWeightsWithTypeAssertion[int]("priority", priorities)
	if err != nil {
		return err
	}
	return s.Initialize(prioritiesInt)
}

type RankedIndex struct {
	Index int
	Rank  int
}

func (s *HighestAlwaysFirst) NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool) {
	res := make([]RankedIndex, 0, upto)
	numDistinct := 0
	prevPriority := 0
	for i := 0; i < len(s.sortedPriorities); i++ {
		if s.sortedPriorities[i].Priority != prevPriority {
			numDistinct++
			if len(res) >= upto {
				return res, len(res) == len(s.sortedPriorities)
			}
			prevPriority = s.sortedPriorities[i].Priority
		}
		res = append(res, RankedIndex{
			Index: s.sortedPriorities[i].OriginalIndex,
			Rank:  numDistinct,
		})
	}
	return res, len(res) == len(s.sortedPriorities)
}

//func (s *HighestAlwaysFirst) NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool) {
//	res := make([]RankedIndex, 0, upto)
//	for i := 0; i < upto && i < len(s.sortedPriorities); i++ {
//		res = append(res, RankedIndex{
//			Index: s.sortedPriorities[i].OriginalIndex,
//			Rank:  1,
//		})
//	}
//	return res, len(res) == len(s.sortedPriorities)
//}

func (s *HighestAlwaysFirst) UpdateOnCaseSelected(index int) {}

//func (s *HighestAlwaysFirst) UpdateOnCaseSelected(index int) {
//	startShuffleIndex, finishShuffleIndex := -1, -1
//	for i := 1; i <= len(s.sortedPriorities)-1; i++ {
//		if startShuffleIndex == -1 && s.sortedPriorities[i].Priority == s.sortedPriorities[i-1].Priority {
//			startShuffleIndex = i - 1
//		}
//		if startShuffleIndex != -1 && s.sortedPriorities[i].Priority != s.sortedPriorities[i-1].Priority {
//			finishShuffleIndex = i - 1
//			shuffle(s.sortedPriorities[startShuffleIndex : finishShuffleIndex+1])
//			startShuffleIndex = -1
//		}
//	}
//	if startShuffleIndex != -1 {
//		shuffle(s.sortedPriorities[startShuffleIndex:])
//	}
//}

//func shuffle(a []sortedToOriginalIndex) {
//	for i := len(a) - 1; i > 0; i-- {
//		j := int(rand.Uint64N(uint64(i + 1)))
//		a[i], a[j] = a[j], a[i]
//	}
//}

func (s *HighestAlwaysFirst) DisableSelectCase(index int) {
	if index < 0 || index > len(s.origPriorities)-1 {
		return
	}
	if _, ok := s.disabledCases[index]; ok {
		return
	}
	priority := s.origPriorities[index]
	spIndex := sort.Search(len(s.sortedPriorities), func(i int) bool {
		spi := s.sortedPriorities[i]
		return (priority > spi.Priority) ||
			(priority == spi.Priority && index >= spi.OriginalIndex)
	})
	if spIndex == len(s.sortedPriorities) || s.sortedPriorities[spIndex].OriginalIndex != index {
		// this should never happen
		return
	}
	s.sortedPriorities = append(s.sortedPriorities[:spIndex], s.sortedPriorities[spIndex+1:]...)
	s.disabledCases[index] = priority
}
