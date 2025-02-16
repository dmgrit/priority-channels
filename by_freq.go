package priority_channels

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels/channels"
)

func NewByFrequencyRatio[T any](ctx context.Context,
	channelsWithFreqRatios []channels.ChannelWithFreqRatio[T],
	options ...func(*PriorityChannelOptions)) (PriorityChannel[T], error) {
	if err := validateInputChannels(convertChannelsWithFreqRatiosToChannels(channelsWithFreqRatios)); err != nil {
		return nil, err
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	return &priorityChannel[T]{
		ctx:                        ctx,
		compositeChannel:           newCompositeChannelByFreqRatio("", channelsWithFreqRatios),
		channelReceiveWaitInterval: pcOptions.channelReceiveWaitInterval,
	}, nil
}

func newCompositeChannelByFreqRatio[T any](name string, channelsWithFreqRatios []channels.ChannelWithFreqRatio[T]) channels.SelectableChannel[T] {
	zeroLevel := &level[T]{}
	zeroLevel.Buckets = make([]*priorityBucket[T], 0, len(channelsWithFreqRatios))
	for _, q := range channelsWithFreqRatios {
		bucket := &priorityBucket[T]{
			Channel: q,
			Value:   0,
		}
		zeroLevel.Buckets = append(zeroLevel.Buckets, bucket)
		zeroLevel.TotalCapacity += bucket.Capacity()
	}
	sort.Slice(zeroLevel.Buckets, func(i int, j int) bool {
		return zeroLevel.Buckets[i].Capacity() > zeroLevel.Buckets[j].Capacity()
	})
	return &compositeChannelByFreqRatio[T]{
		channelName:  name,
		levels:       []*level[T]{zeroLevel},
		totalBuckets: len(channelsWithFreqRatios),
	}
}

type priorityBucket[T any] struct {
	Channel channels.ChannelWithFreqRatio[T]
	Value   int
}

func (pb *priorityBucket[T]) ChannelName() string {
	return pb.Channel.ChannelName()
}

func (pb *priorityBucket[T]) Capacity() int {
	return pb.Channel.FreqRatio()
}

type level[T any] struct {
	TotalValue    int
	TotalCapacity int
	Buckets       []*priorityBucket[T]
}

type compositeChannelByFreqRatio[T any] struct {
	channelName  string
	levels       []*level[T]
	totalBuckets int
}

func (c *compositeChannelByFreqRatio[T]) ChannelName() string {
	return c.channelName
}

func (c *compositeChannelByFreqRatio[T]) NextSelectCases(upto int) ([]channels.SelectCase[T], bool, *channels.ClosedChannelDetails) {
	addedBuckets := 0
	numOfBucketsToProcess := upto
	currIndex := 0
	selectCases := make([]channels.SelectCase[T], 0, numOfBucketsToProcess)
	areAllCasesAdded := false
	for i, level := range c.levels {
		for j, b := range level.Buckets {
			channelSelectCases, allSelected, closedChannel := b.Channel.NextSelectCases(upto - addedBuckets)
			if closedChannel != nil {
				return nil, true, &channels.ClosedChannelDetails{
					ChannelName: closedChannel.ChannelName,
					PathInTree: append(closedChannel.PathInTree, channels.ChannelNode{
						ChannelName:  c.channelName,
						ChannelIndex: i,
					}),
				}
			}
			for _, sc := range channelSelectCases {
				selectCases = append(selectCases, channels.SelectCase[T]{
					MsgsC:       sc.MsgsC,
					ChannelName: sc.ChannelName,
					PathInTree: append(sc.PathInTree, channels.ChannelNode{
						ChannelName:  c.channelName,
						ChannelIndex: currIndex,
					}),
				})
				addedBuckets++
				areAllCasesAdded = (i == len(c.levels)-1) && (j == len(level.Buckets)-1) && allSelected
				if addedBuckets == numOfBucketsToProcess {
					return selectCases, areAllCasesAdded, nil
				}
			}
			currIndex++
		}
	}
	return selectCases, areAllCasesAdded, nil
}

func (c *compositeChannelByFreqRatio[T]) UpdateOnCaseSelected(pathInTree []channels.ChannelNode) {
	if len(pathInTree) == 0 {
		return
	}
	index := pathInTree[len(pathInTree)-1].ChannelIndex
	levelIndex, bucketIndex := c.getLevelAndBucketIndexByChosenChannelIndex(index)
	chosenLevel := c.levels[levelIndex]
	chosenBucket := chosenLevel.Buckets[bucketIndex]
	chosenBucket.Channel.UpdateOnCaseSelected(pathInTree[:len(pathInTree)-1])

	c.updateStateOnReceivingMessageToBucket(levelIndex, bucketIndex)
}

func (c *compositeChannelByFreqRatio[T]) Validate() error {
	return nil
}

func (c *compositeChannelByFreqRatio[T]) getLevelAndBucketIndexByChosenChannelIndex(chosen int) (levelIndex int, bucketIndex int) {
	currIndex := 0
	for i := range c.levels {
		for j := range c.levels[i].Buckets {
			if currIndex == chosen {
				return i, j
			}
			currIndex++
		}
	}
	return -1, -1
}

func (c *compositeChannelByFreqRatio[T]) updateStateOnReceivingMessageToBucket(levelIndex int, bucketIndex int) {
	chosenLevel := c.levels[levelIndex]
	chosenBucket := chosenLevel.Buckets[bucketIndex]
	chosenBucket.Value++
	chosenLevel.TotalValue++

	if chosenLevel.TotalValue == chosenLevel.TotalCapacity {
		c.mergeAllNextLevelsBackIntoCurrentLevel(levelIndex)
		return
	}
	if chosenBucket.Value == chosenBucket.Capacity() {
		c.moveBucketToNextLevel(levelIndex, bucketIndex)
		return
	}
}

func (c *compositeChannelByFreqRatio[T]) mergeAllNextLevelsBackIntoCurrentLevel(levelIndex int) {
	chosenLevel := c.levels[levelIndex]
	if levelIndex < len(c.levels)-1 {
		for nextLevelIndex := levelIndex + 1; nextLevelIndex <= len(c.levels)-1; nextLevelIndex++ {
			nextLevel := c.levels[nextLevelIndex]
			chosenLevel.Buckets = append(chosenLevel.Buckets, nextLevel.Buckets...)
		}
		sort.Slice(chosenLevel.Buckets, func(i int, j int) bool {
			return chosenLevel.Buckets[i].Capacity() > chosenLevel.Buckets[j].Capacity()
		})
		c.levels = c.levels[0 : levelIndex+1]
	}
	chosenLevel.TotalValue = 0
	for i := range chosenLevel.Buckets {
		chosenLevel.Buckets[i].Value = 0
	}
}

func (c *compositeChannelByFreqRatio[T]) moveBucketToNextLevel(levelIndex int, bucketIndex int) {
	chosenLevel := c.levels[levelIndex]
	chosenBucket := chosenLevel.Buckets[bucketIndex]
	chosenBucket.Value = 0
	if len(chosenLevel.Buckets) == 1 {
		// if this bucket is the only one on its level - no need to move it to next level
		chosenLevel.TotalValue = 0
		return
	}
	if levelIndex == len(c.levels)-1 {
		c.levels = append(c.levels, &level[T]{})
	}
	nextLevel := c.levels[levelIndex+1]
	nextLevel.TotalCapacity += chosenBucket.Capacity()
	chosenLevel.Buckets = append(chosenLevel.Buckets[:bucketIndex], chosenLevel.Buckets[bucketIndex+1:]...)
	i := sort.Search(len(nextLevel.Buckets), func(i int) bool {
		return nextLevel.Buckets[i].Capacity() < chosenBucket.Capacity()
	})
	nextLevel.Buckets = append(nextLevel.Buckets, &priorityBucket[T]{})
	copy(nextLevel.Buckets[i+1:], nextLevel.Buckets[i:])
	nextLevel.Buckets[i] = chosenBucket
}
