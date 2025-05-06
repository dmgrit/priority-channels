package frequency_strategies

import (
	"github.com/dmgrit/priority-channels/internal/collections"
	"github.com/dmgrit/priority-channels/strategies"
)

type WithStrictOrderAcrossCycles struct {
	levels            collections.DoublyLinkedList[*aLevel]
	origIndexToBucket map[int]*collections.ListNode[*aPriorityBucket]
	disabledCases     map[int]int
}

type aPriorityBucket struct {
	Value            int
	Capacity         int
	Level            *collections.ListNode[*aLevel]
	OrigChannelIndex int
}

type aLevel struct {
	Buckets collections.DoublyLinkedList[*aPriorityBucket]
}

func NewWithStrictOrderAcrossCycles() *WithStrictOrderAcrossCycles {
	return &WithStrictOrderAcrossCycles{
		origIndexToBucket: make(map[int]*collections.ListNode[*aPriorityBucket]),
		disabledCases:     make(map[int]int),
	}
}

func (s *WithStrictOrderAcrossCycles) Initialize(freqRatios []int) error {
	zeroLevel := collections.NewListNode(&aLevel{})
	for i, freqRatio := range freqRatios {
		if freqRatio <= 0 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          ErrFreqRatioMustBeGreaterThanZero,
			}
		}
		bucket := &aPriorityBucket{
			OrigChannelIndex: i,
			Value:            0,
			Capacity:         freqRatio,
			Level:            zeroLevel,
		}
		bucketNode := collections.NewListNode(bucket)
		s.origIndexToBucket[i] = bucketNode
		zeroLevel.Value().Buckets.Append(bucketNode)
	}
	s.levels.Append(zeroLevel)
	return nil
}

func (s *WithStrictOrderAcrossCycles) InitializeWithTypeAssertion(freqRatios []interface{}) error {
	freqRatiosInt, err := strategies.ConvertWeightsWithTypeAssertion[int]("frequency ratio", freqRatios)
	if err != nil {
		return err
	}
	return s.Initialize(freqRatiosInt)
}

func (s *WithStrictOrderAcrossCycles) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	res := make([]strategies.RankedIndex, 0, upto)
	rank := 0

	it := s.levels.Iterator()
	for it.HasNext() {
		rank++
		level := it.Next()
		bucketsIterator := level.Buckets.Iterator()
		for bucketsIterator.HasNext() {
			b := bucketsIterator.Next()
			res = append(res, strategies.RankedIndex{Index: b.OrigChannelIndex, Rank: rank})
		}
		if len(res) >= upto {
			return res, !it.HasNext()
		}
	}
	return res, true
}

func (s *WithStrictOrderAcrossCycles) UpdateOnCaseSelected(index int) {
	bucket := s.origIndexToBucket[index]
	s.updateStateOnReceivingMessageToBucket(bucket)
}

func (s *WithStrictOrderAcrossCycles) DisableSelectCase(index int) {
	if _, ok := s.disabledCases[index]; ok {
		return
	}
	bucket, ok := s.origIndexToBucket[index]
	if !ok {
		return
	}
	s.removeBucketFromItsLevel(bucket)
	delete(s.origIndexToBucket, index)
	s.disabledCases[index] = bucket.Value().Capacity
}

func (s *WithStrictOrderAcrossCycles) EnableSelectCase(index int) {
	freqRatio, ok := s.disabledCases[index]
	if !ok {
		return
	}
	delete(s.disabledCases, index)
	bucket := &aPriorityBucket{
		OrigChannelIndex: index,
		Value:            0,
		Capacity:         freqRatio,
	}
	bucketNode := collections.NewListNode(bucket)
	s.origIndexToBucket[index] = bucketNode
	firstLevel := s.levels.FirstNode()
	if firstLevel == nil {
		s.levels.Append(collections.NewListNode(&aLevel{}))
		firstLevel = s.levels.FirstNode()
	}
	s.addBucketToLevel(bucketNode, firstLevel)
}

func (s *WithStrictOrderAcrossCycles) updateStateOnReceivingMessageToBucket(bucket *collections.ListNode[*aPriorityBucket]) {
	b := bucket.Value()
	b.Value++

	if b.Value != b.Capacity {
		return
	}
	b.Value = 0
	s.moveBucketToLastLevel(bucket)
}

func (s *WithStrictOrderAcrossCycles) moveBucketToLastLevel(bucket *collections.ListNode[*aPriorityBucket]) {
	isNeeded := s.prepareToMovingBucketIfNeeded(bucket)
	if !isNeeded {
		return
	}

	s.removeBucketFromItsLevel(bucket)
	// add bucket to the last level
	s.addBucketToLevel(bucket, s.levels.LastNode())
}

func (s *WithStrictOrderAcrossCycles) addBucketToLevel(bucket *collections.ListNode[*aPriorityBucket], dstLevel *collections.ListNode[*aLevel]) {
	dstLevel.Value().Buckets.Append(bucket)
	bucket.Value().Level = dstLevel
}

func (s *WithStrictOrderAcrossCycles) prepareToMovingBucketIfNeeded(bucket *collections.ListNode[*aPriorityBucket]) bool {
	level := bucket.Value().Level
	if level.HasNext() {
		// if bucket is not in the last level, we need to move it
		return true
	}
	if level.Value().Buckets.Len() == 1 {
		// bucket is currently in the last level,
		// and it is the only one in the level, no need to move it
		return false
	}
	// bucket is currently in the last level, and there are other buckets in the level
	// Add a new level for adding the bucket to it
	s.levels.Append(collections.NewListNode(&aLevel{}))
	return true
}

func (s *WithStrictOrderAcrossCycles) removeBucketFromItsLevel(bucket *collections.ListNode[*aPriorityBucket]) {
	b := bucket.Value()
	levelBuckets := &b.Level.Value().Buckets
	if levelBuckets.Len() == 1 {
		s.levels.RemoveNode(b.Level)
	} else {
		levelBuckets.RemoveNode(bucket)
	}
	b.Level = nil
	b.Value = 0
}
