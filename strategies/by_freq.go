package strategies

import (
	"errors"
	"sort"
)

var ErrFreqRatioMustBeGreaterThanZero = errors.New("frequency ratio must be greater than 0")

type ByFreqRatio struct {
	channelName       string
	levels            []*level
	origIndexToBucket map[int]*priorityBucket
}

func NewByFreqRatio() *ByFreqRatio {
	return &ByFreqRatio{}
}

func byFreqPriorityBucketsSortingFunc(b1, b2 *priorityBucket) bool {
	return (b1.Capacity > b2.Capacity) ||
		(b1.Capacity == b2.Capacity && b1.OrigChannelIndex > b2.OrigChannelIndex)
}

func (s *ByFreqRatio) Initialize(freqRatios []int) error {
	zeroLevel := &level{}
	zeroLevel.Buckets = make([]*priorityBucket, 0, len(freqRatios))
	s.origIndexToBucket = make(map[int]*priorityBucket)
	for i, freqRatio := range freqRatios {
		if freqRatio <= 0 {
			return &WeightValidationError{
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

func (s *ByFreqRatio) InitializeWithTypeAssertion(freqRatios []interface{}) error {
	freqRatiosInt, err := convertWeightsWithTypeAssertion[int]("frequency ratio", freqRatios)
	if err != nil {
		return err
	}
	return s.Initialize(freqRatiosInt)
}

func (s *ByFreqRatio) NextSelectCasesIndexes(upto int) []int {
	res := make([]int, 0, upto)
	for _, level := range s.levels {
		for _, b := range level.Buckets {
			res = append(res, b.OrigChannelIndex)
			if len(res) == upto {
				return res
			}
		}
	}
	return res
}

func (s *ByFreqRatio) UpdateOnCaseSelected(index int) {
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

type priorityBucket struct {
	Value            int
	Capacity         int
	LevelIndex       int
	OrigChannelIndex int
}

type level struct {
	Buckets []*priorityBucket
}

func (s *ByFreqRatio) updateStateOnReceivingMessageToBucket(levelIndex int, bucketIndex int) {
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

func (s *ByFreqRatio) moveBucketToLastLevel(levelIndex int, bucketIndex int) {
	if levelIndex == len(s.levels)-1 {
		if len(s.levels[levelIndex].Buckets) == 1 {
			// if this bucket is currently in the last level,
			// and it is the only one in the level, no need to move it
			return
		} else {
			// the bucket is currently on the last level, and there are other buckets in the level
			// Add a new level and move the bucket to it
			s.levels = append(s.levels, &level{})
		}
	}

	srcLevel := s.levels[levelIndex]
	bucket := srcLevel.Buckets[bucketIndex]
	// remove bucket from its current level
	srcLevel.Buckets = append(srcLevel.Buckets[:bucketIndex], srcLevel.Buckets[bucketIndex+1:]...)

	// add bucket to the correct position in last level
	lastLevel := s.levels[len(s.levels)-1]
	i := sort.Search(len(lastLevel.Buckets), func(i int) bool {
		return (bucket.Capacity > lastLevel.Buckets[i].Capacity) ||
			(bucket.Capacity == lastLevel.Buckets[i].Capacity && bucket.OrigChannelIndex > lastLevel.Buckets[i].OrigChannelIndex)
	})
	lastLevel.Buckets = append(lastLevel.Buckets, &priorityBucket{})
	copy(lastLevel.Buckets[i+1:], lastLevel.Buckets[i:])
	lastLevel.Buckets[i] = bucket
	bucket.LevelIndex = len(s.levels) - 1
}

func (c *ByFreqRatio) removeEmptyLevel(levelIndex int) {
	c.levels = append(c.levels[:levelIndex], c.levels[levelIndex+1:]...)

	// Fix level index for all buckets in levels after the removed level
	for i := levelIndex; i < len(c.levels); i++ {
		for _, bucket := range c.levels[i].Buckets {
			bucket.LevelIndex = i
		}
	}
}
