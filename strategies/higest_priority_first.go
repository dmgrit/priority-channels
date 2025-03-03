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
	prioritiesInt, err := convertWeightsWithTypeAssertion[int]("priority", priorities)
	if err != nil {
		return err
	}
	return s.Initialize(prioritiesInt)
}

func (s *HighestAlwaysFirst) NextSelectCasesIndexes(upto int) []int {
	res := make([]int, 0, upto)
	for i := 0; i < upto && i < len(s.sortedPriorities); i++ {
		res = append(res, s.sortedPriorities[i].OriginalIndex)
	}
	return res
}

func (s *HighestAlwaysFirst) UpdateOnCaseSelected(index int) {}

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
