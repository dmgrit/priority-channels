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
		return zeroLevel.Buckets[i].Capacity > zeroLevel.Buckets[j].Capacity
	})
	for i, bucket := range zeroLevel.Buckets {
		bucket.BucketIndex = i
	}

	s.levels = []*level{zeroLevel}
	s.totalBuckets = len(freqRatios)
	return nil
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
	s.updateStateOnReceivingMessageToBucket(bucket.LevelIndex, bucket.BucketIndex)
}

type priorityBucket struct {
	Value            int
	Capacity         int
	LevelIndex       int
	BucketIndex      int
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
			return chosenLevel.Buckets[i].Capacity > chosenLevel.Buckets[j].Capacity
		})
		c.levels = c.levels[0 : levelIndex+1]
	}
	chosenLevel.TotalValue = 0
	for i, bucket := range chosenLevel.Buckets {
		bucket.Value = 0
		bucket.LevelIndex = levelIndex
		bucket.BucketIndex = i
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
	for j := bucketIndex; j < len(chosenLevel.Buckets); j++ {
		chosenLevel.Buckets[j].BucketIndex = j
	}
	i := sort.Search(len(nextLevel.Buckets), func(i int) bool {
		return nextLevel.Buckets[i].Capacity < chosenBucket.Capacity
	})
	nextLevel.Buckets = append(nextLevel.Buckets, &priorityBucket{})
	copy(nextLevel.Buckets[i+1:], nextLevel.Buckets[i:])
	nextLevel.Buckets[i] = chosenBucket
	chosenBucket.LevelIndex = levelIndex + 1
	chosenBucket.BucketIndex = i
	for j := i + 1; j < len(nextLevel.Buckets); j++ {
		nextLevel.Buckets[j].BucketIndex = j
	}
}
