package priority_channels

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/internal/selectable"
)

func NewByHighestAlwaysFirst[T any](ctx context.Context,
	channelsWithPriorities []channels.ChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	selectableChannels := make([]selectable.ChannelWithPriority[T], 0, len(channelsWithPriorities))
	for _, c := range channelsWithPriorities {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithPriority(c))
	}
	return newByHighestAlwaysFirst(ctx, selectableChannels, options...)
}

func newByHighestAlwaysFirst[T any](ctx context.Context,
	channelsWithPriorities []selectable.ChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	if err := validateInputChannels(convertChannelsWithPrioritiesToChannels(channelsWithPriorities)); err != nil {
		return nil, err
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	return &PriorityChannel[T]{
		ctx:                        ctx,
		compositeChannel:           newCompositeChannelByPriority("", channelsWithPriorities),
		channelReceiveWaitInterval: pcOptions.channelReceiveWaitInterval,
	}, nil
}

func newCompositeChannelByPriority[T any](name string, channelsWithPriorities []selectable.ChannelWithPriority[T]) selectable.Channel[T] {
	ch := &compositeChannelByPriority[T]{
		channelName: name,
		channels:    make([]selectable.ChannelWithPriority[T], 0, len(channelsWithPriorities)),
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
	channels    []selectable.ChannelWithPriority[T]
}

func (c *compositeChannelByPriority[T]) ChannelName() string {
	return c.channelName
}

func (c *compositeChannelByPriority[T]) NextSelectCases(upto int) ([]selectable.SelectCase[T], bool, *selectable.ClosedChannelDetails) {
	added := 0
	var selectCases []selectable.SelectCase[T]
	areAllCasesAdded := false
	for i := 0; i <= len(c.channels)-1; i++ {
		channelSelectCases, allSelected, closedChannel := c.channels[i].NextSelectCases(upto - added)
		if closedChannel != nil {
			return nil, true, &selectable.ClosedChannelDetails{
				ChannelName: closedChannel.ChannelName,
				PathInTree: append(closedChannel.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: i,
				}),
			}
		}
		for _, sc := range channelSelectCases {
			selectCases = append(selectCases, selectable.SelectCase[T]{
				ChannelName: sc.ChannelName,
				MsgsC:       sc.MsgsC,
				PathInTree: append(sc.PathInTree, selectable.ChannelNode{
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

func (c *compositeChannelByPriority[T]) UpdateOnCaseSelected(pathInTree []selectable.ChannelNode) {
	if len(pathInTree) == 0 {
		return
	}
	selectedChannel := c.channels[pathInTree[len(pathInTree)-1].ChannelIndex]
	selectedChannel.UpdateOnCaseSelected(pathInTree[:len(pathInTree)-1])
}

func (c *compositeChannelByPriority[T]) Validate() error {
	return nil
}
