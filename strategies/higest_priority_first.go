package strategies

import (
	"errors"
	"sort"
)

var ErrPriorityIsNegative = errors.New("priority cannot be negative")

type HighestAlwaysFirst struct {
	sortedPriorities []sortedToOriginalIndex
}

type sortedToOriginalIndex struct {
	Priority      int
	OriginalIndex int
}

func NewByHighestAlwaysFirst() *HighestAlwaysFirst {
	return &HighestAlwaysFirst{}
}

func (s *HighestAlwaysFirst) Initialize(priorities []int) error {
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
		return s.sortedPriorities[i].Priority > s.sortedPriorities[j].Priority
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
