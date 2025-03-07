package frequency_strategies

import (
	"errors"
	"sort"

	"github.com/dmgrit/priority-channels/strategies"
)

var ErrFreqRatioMustBeGreaterThanZero = errors.New("frequency ratio must be greater than 0")

type WithStrictOrder struct {
	levels            []*level
	origIndexToBucket map[int]*priorityBucket
	disabledCases     map[int]int
	fully             bool
}

func NewWithStrictOrderAcrossCycles() *WithStrictOrder {
	return newWithStrictOrder(false)
}

func NewWithStrictOrderFully() *WithStrictOrder {
	return newWithStrictOrder(true)
}

func newWithStrictOrder(fully bool) *WithStrictOrder {
	return &WithStrictOrder{
		origIndexToBucket: make(map[int]*priorityBucket),
		disabledCases:     make(map[int]int),
		fully:             fully,
	}
}

func byFreqPriorityBucketsSortingFunc(b1, b2 *priorityBucket) bool {
	return (b1.Capacity > b2.Capacity) ||
		(b1.Capacity == b2.Capacity && b1.OrigChannelIndex > b2.OrigChannelIndex)
}

func (s *WithStrictOrder) Initialize(freqRatios []int) error {
	zeroLevel := &level{}
	zeroLevel.Buckets = make([]*priorityBucket, 0, len(freqRatios))
	for i, freqRatio := range freqRatios {
		if freqRatio <= 0 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          ErrFreqRatioMustBeGreaterThanZero,
			}
		}
		bucket := &priorityBucket{
			OrigChannelIndex: i,
			Value:            0,
			Capacity:         freqRatio,
		}
		s.origIndexToBucket[i] = bucket
		zeroLevel.Buckets = append(zeroLevel.Buckets, bucket)
	}
	sort.Slice(zeroLevel.Buckets, func(i int, j int) bool {
		return byFreqPriorityBucketsSortingFunc(zeroLevel.Buckets[i], zeroLevel.Buckets[j])
	})

	s.levels = []*level{zeroLevel}
	return nil
}

func (s *WithStrictOrder) InitializeWithTypeAssertion(freqRatios []interface{}) error {
	freqRatiosInt, err := strategies.ConvertWeightsWithTypeAssertion[int]("frequency ratio", freqRatios)
	if err != nil {
		return err
	}
	return s.Initialize(freqRatiosInt)
}

func (s *WithStrictOrder) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	if s.fully {
		return s.nextSelectCasesRankedIndexesWithStrictOrder(upto)
	}
	return s.nextSelectCasesRankedIndexesWithoutStrictOrder(upto)
}

func (s *WithStrictOrder) nextSelectCasesRankedIndexesWithStrictOrder(upto int) ([]strategies.RankedIndex, bool) {
	res := make([]strategies.RankedIndex, 0, upto)
	rank := 0
	for i, level := range s.levels {
		for j, b := range level.Buckets {
			rank++
			res = append(res, strategies.RankedIndex{Index: b.OrigChannelIndex, Rank: rank})
			if len(res) == upto {
				return res, i == len(s.levels)-1 && j == len(level.Buckets)-1
			}
		}
	}
	return res, true
}

func (s *WithStrictOrder) nextSelectCasesRankedIndexesWithoutStrictOrder(upto int) ([]strategies.RankedIndex, bool) {
	res := make([]strategies.RankedIndex, 0, upto)
	rank := 0
	for i, level := range s.levels {
		rank++
		j := 0
		for _, b := range level.Buckets {
			j++
			res = append(res, strategies.RankedIndex{Index: b.OrigChannelIndex, Rank: rank})
		}
		if len(res) >= upto {
			return res, i == len(s.levels)-1 && j == len(level.Buckets)-1
		}
	}
	return res, true
}

func (s *WithStrictOrder) UpdateOnCaseSelected(index int) {
	bucket := s.origIndexToBucket[index]
	levelBuckets := s.levels[bucket.LevelIndex].Buckets
	bucketIndex := sort.Search(len(levelBuckets), func(i int) bool {
		return (bucket.Capacity > levelBuckets[i].Capacity) ||
			(bucket.Capacity == levelBuckets[i].Capacity && bucket.OrigChannelIndex >= levelBuckets[i].OrigChannelIndex)
	})
	if bucketIndex == len(levelBuckets) || levelBuckets[bucketIndex].OrigChannelIndex != index {
		// this should never happen
		return
	}
	s.updateStateOnReceivingMessageToBucket(bucket.LevelIndex, bucketIndex)
}

func (s *WithStrictOrder) DisableSelectCase(index int) {
	if _, ok := s.disabledCases[index]; ok {
		return
	}
	bucket, ok := s.origIndexToBucket[index]
	if !ok {
		return
	}
	levelBuckets := s.levels[bucket.LevelIndex].Buckets
	bucketIndex := sort.Search(len(levelBuckets), func(i int) bool {
		return (bucket.Capacity > levelBuckets[i].Capacity) ||
			(bucket.Capacity == levelBuckets[i].Capacity && bucket.OrigChannelIndex >= levelBuckets[i].OrigChannelIndex)
	})
	if bucketIndex == len(levelBuckets) || levelBuckets[bucketIndex].OrigChannelIndex != index {
		// This should not happen
		return
	}
	s.removeBucket(bucket.LevelIndex, bucketIndex)
	delete(s.origIndexToBucket, index)
	s.disabledCases[index] = bucket.Capacity
}

type priorityBucket struct {
	Value            int
	Capacity         int
	LevelIndex       int
	OrigChannelIndex int
}

type level struct {
	Buckets []*priorityBucket
}

func (s *WithStrictOrder) updateStateOnReceivingMessageToBucket(levelIndex int, bucketIndex int) {
	chosenLevel := s.levels[levelIndex]
	chosenBucket := chosenLevel.Buckets[bucketIndex]
	chosenBucket.Value++

	if chosenBucket.Value != chosenBucket.Capacity {
		return
	}
	chosenBucket.Value = 0
	s.moveBucketToLastLevel(levelIndex, bucketIndex)

	// if after moving the bucket to the last level the current level is empty, remove it
	if len(chosenLevel.Buckets) == 0 {
		s.removeEmptyLevel(levelIndex)
	}
}

func (s *WithStrictOrder) moveBucketToLastLevel(levelIndex int, bucketIndex int) {
	isNeeded := s.prepareToMovingBucketIfNeeded(levelIndex)
	if !isNeeded {
		return
	}

	srcLevel := s.levels[levelIndex]
	bucket := srcLevel.Buckets[bucketIndex]
	// remove bucket from its current level
	srcLevel.Buckets = append(srcLevel.Buckets[:bucketIndex], srcLevel.Buckets[bucketIndex+1:]...)

	// add bucket to the correct position in last level
	s.addBucketToLevel(bucket, len(s.levels)-1)
}

func (s *WithStrictOrder) addBucketToLevel(bucket *priorityBucket, levelIndex int) {
	dstLevel := s.levels[levelIndex]
	i := sort.Search(len(dstLevel.Buckets), func(i int) bool {
		return (bucket.Capacity > dstLevel.Buckets[i].Capacity) ||
			(bucket.Capacity == dstLevel.Buckets[i].Capacity && bucket.OrigChannelIndex > dstLevel.Buckets[i].OrigChannelIndex)
	})
	dstLevel.Buckets = append(dstLevel.Buckets, &priorityBucket{})
	copy(dstLevel.Buckets[i+1:], dstLevel.Buckets[i:])
	dstLevel.Buckets[i] = bucket
	bucket.LevelIndex = levelIndex
}

func (s *WithStrictOrder) prepareToMovingBucketIfNeeded(levelIndex int) bool {
	if levelIndex != len(s.levels)-1 {
		// if bucket is not in the last level, we need to move it
		return true
	}
	if len(s.levels[levelIndex].Buckets) == 1 {
		// bucket is currently in the last level,
		// and it is the only one in the level, no need to move it
		return false
	}
	// bucket is currently in the last level, and there are other buckets in the level
	// Add a new level for adding the bucket to it
	s.levels = append(s.levels, &level{})
	return true
}

func (s *WithStrictOrder) removeEmptyLevel(levelIndex int) {
	s.levels = append(s.levels[:levelIndex], s.levels[levelIndex+1:]...)

	// Fix level index for all buckets in levels after the removed level
	for i := levelIndex; i < len(s.levels); i++ {
		for _, bucket := range s.levels[i].Buckets {
			bucket.LevelIndex = i
		}
	}
}

func (c *WithStrictOrder) removeBucket(levelIndex int, bucketIndex int) {
	chosenBucket := c.levels[levelIndex].Buckets[bucketIndex]
	if len(c.levels[levelIndex].Buckets) == 1 {
		c.removeEmptyLevel(levelIndex)
	} else {
		// remove bucket in level
		c.levels[levelIndex].Buckets = append(
			c.levels[levelIndex].Buckets[:bucketIndex],
			c.levels[levelIndex].Buckets[bucketIndex+1:]...)
	}
	chosenBucket.LevelIndex = -1
	chosenBucket.Value = 0
}
