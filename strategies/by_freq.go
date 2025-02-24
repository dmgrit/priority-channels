package strategies

import (
	"errors"
	"sort"
)

var ErrFreqRatioMustBeGreaterThanZero = errors.New("frequency ratio must be greater than 0")

type ByFreqRatio struct {
	channelName       string
	levels            []*level
	totalBuckets      int
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
		zeroLevel.TotalCapacity += bucket.Capacity
	}
	sort.Slice(zeroLevel.Buckets, func(i int, j int) bool {
		return byFreqPriorityBucketsSortingFunc(zeroLevel.Buckets[i], zeroLevel.Buckets[j])
	})

	s.levels = []*level{zeroLevel}
	s.totalBuckets = len(freqRatios)
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
	TotalValue    int
	TotalCapacity int
	Buckets       []*priorityBucket
}

func (c *ByFreqRatio) updateStateOnReceivingMessageToBucket(levelIndex int, bucketIndex int) {
	chosenLevel := c.levels[levelIndex]
	chosenBucket := chosenLevel.Buckets[bucketIndex]
	chosenBucket.Value++
	chosenLevel.TotalValue++

	if chosenLevel.TotalValue == chosenLevel.TotalCapacity {
		c.mergeAllNextLevelsBackIntoCurrentLevel(levelIndex)
		return
	}
	if chosenBucket.Value == chosenBucket.Capacity {
		c.moveBucketToNextLevel(levelIndex, bucketIndex)
		return
	}
}

func (c *ByFreqRatio) mergeAllNextLevelsBackIntoCurrentLevel(levelIndex int) {
	chosenLevel := c.levels[levelIndex]
	if levelIndex < len(c.levels)-1 {
		for nextLevelIndex := levelIndex + 1; nextLevelIndex <= len(c.levels)-1; nextLevelIndex++ {
			nextLevel := c.levels[nextLevelIndex]
			chosenLevel.Buckets = append(chosenLevel.Buckets, nextLevel.Buckets...)
		}
		sort.Slice(chosenLevel.Buckets, func(i int, j int) bool {
			return byFreqPriorityBucketsSortingFunc(chosenLevel.Buckets[i], chosenLevel.Buckets[j])
		})
		c.levels = c.levels[0 : levelIndex+1]
	}
	chosenLevel.TotalValue = 0
	for _, bucket := range chosenLevel.Buckets {
		bucket.Value = 0
		bucket.LevelIndex = levelIndex
	}
}

func (c *ByFreqRatio) moveBucketToNextLevel(levelIndex int, bucketIndex int) {
	chosenLevel := c.levels[levelIndex]
	chosenBucket := chosenLevel.Buckets[bucketIndex]
	chosenBucket.Value = 0
	if len(chosenLevel.Buckets) == 1 {
		// if this bucket is the only one on its level - no need to move it to next level
		chosenLevel.TotalValue = 0
		return
	}
	if levelIndex == len(c.levels)-1 {
		c.levels = append(c.levels, &level{})
	}
	nextLevel := c.levels[levelIndex+1]
	nextLevel.TotalCapacity += chosenBucket.Capacity
	chosenLevel.Buckets = append(chosenLevel.Buckets[:bucketIndex], chosenLevel.Buckets[bucketIndex+1:]...)
	i := sort.Search(len(nextLevel.Buckets), func(i int) bool {
		return (chosenBucket.Capacity > nextLevel.Buckets[i].Capacity) ||
			(chosenBucket.Capacity == nextLevel.Buckets[i].Capacity && chosenBucket.OrigChannelIndex > nextLevel.Buckets[i].OrigChannelIndex)
	})
	nextLevel.Buckets = append(nextLevel.Buckets, &priorityBucket{})
	copy(nextLevel.Buckets[i+1:], nextLevel.Buckets[i:])
	nextLevel.Buckets[i] = chosenBucket
	chosenBucket.LevelIndex = levelIndex + 1
}
