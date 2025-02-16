package priority_channels

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels/channels"
)

func NewByHighestAlwaysFirst[T any](ctx context.Context,
	channelsWithPriorities []channels.ChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (PriorityChannel[T], error) {
	if err := validateInputChannels(convertChannelsWithPrioritiesToChannels(channelsWithPriorities)); err != nil {
		return nil, err
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	return &priorityChannel[T]{
		ctx:                        ctx,
		compositeChannel:           newCompositeChannelByPriority("", channelsWithPriorities),
		channelReceiveWaitInterval: pcOptions.channelReceiveWaitInterval,
	}, nil
}

func newCompositeChannelByPriority[T any](name string, channelsWithPriorities []channels.ChannelWithPriority[T]) channels.SelectableChannel[T] {
	ch := &compositeChannelByPriority[T]{
		channelName: name,
		channels:    make([]channels.ChannelWithPriority[T], 0, len(channelsWithPriorities)),
	}
	for _, c := range channelsWithPriorities {
		ch.channels = append(ch.channels, c)
	}
	sort.Slice(ch.channels, func(i int, j int) bool {
		return ch.channels[i].Priority() > ch.channels[j].Priority()
	})
	return ch
}

type compositeChannelByPriority[T any] struct {
	channelName string
	channels    []channels.ChannelWithPriority[T]
}

func (c *compositeChannelByPriority[T]) ChannelName() string {
	return c.channelName
}

func (c *compositeChannelByPriority[T]) NextSelectCases(upto int) ([]channels.SelectCase[T], bool, *channels.ClosedChannelDetails) {
	added := 0
	var selectCases []channels.SelectCase[T]
	areAllCasesAdded := false
	for i := 0; i <= len(c.channels)-1; i++ {
		channelSelectCases, allSelected, closedChannel := c.channels[i].NextSelectCases(upto - added)
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
				ChannelName: sc.ChannelName,
				MsgsC:       sc.MsgsC,
				PathInTree: append(sc.PathInTree, channels.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: i,
				}),
			})
			added++
			areAllCasesAdded = (i == len(c.channels)-1) && allSelected
			if added == upto {
				return selectCases, areAllCasesAdded, nil
			}
		}
	}
	return selectCases, areAllCasesAdded, nil
}

func (c *compositeChannelByPriority[T]) UpdateOnCaseSelected(pathInTree []channels.ChannelNode) {
	if len(pathInTree) == 0 {
		return
	}
	selectedChannel := c.channels[pathInTree[len(pathInTree)-1].ChannelIndex]
	selectedChannel.UpdateOnCaseSelected(pathInTree[:len(pathInTree)-1])
}

func (c *compositeChannelByPriority[T]) Validate() error {
	return nil
}
