package priority_channels

import (
	"context"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies"
)

func NewByStrategy[T any, W any](ctx context.Context,
	strategy PrioritizationStrategy[W],
	channelsWithWeights []channels.ChannelWithWeight[T, W],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	selectableChannels := make([]selectable.ChannelWithWeight[T, W], 0, len(channelsWithWeights))
	for _, c := range channelsWithWeights {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithWeight(c))
	}
	return newByStrategy(ctx, strategy, selectableChannels, options...)
}

type PrioritizationStrategy[W any] interface {
	Initialize(weights []W) error
	NextSelectCasesIndexes(upto int) []int
	UpdateOnCaseSelected(index int)
	DisableSelectCase(index int)
}

func newByStrategy[T any, W any](ctx context.Context,
	strategy PrioritizationStrategy[W],
	channelsWithWeights []selectable.ChannelWithWeight[T, W],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	if err := validateInputChannels(convertChannelsWithWeightsToChannels(channelsWithWeights)); err != nil {
		return nil, err
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	compositeChannel, err := newCompositeChannelByStrategy("", strategy, channelsWithWeights, pcOptions.autoDisableClosedChannels)
	if err != nil {
		return nil, err
	}
	return newPriorityChannel(ctx, compositeChannel, options...), nil
}

func newCompositeChannelByStrategy[T any, W any](name string,
	strategy PrioritizationStrategy[W],
	channelsWithWeights []selectable.ChannelWithWeight[T, W],
	autoDisableClosedChannels bool) (selectable.Channel[T], error) {
	weights := make([]W, 0, len(channelsWithWeights))
	for _, c := range channelsWithWeights {
		weights = append(weights, c.Weight())
	}
	if err := strategy.Initialize(weights); err != nil {
		if wve, ok := err.(*strategies.WeightValidationError); ok {
			invalidChannelName := channelsWithWeights[wve.ChannelIndex].ChannelName()
			return nil, &ChannelValidationError{ChannelName: invalidChannelName, Err: wve.Err}
		}
		return nil, err
	}
	ch := &compositeChannelByPrioritization[T, W]{
		channelName:               name,
		channels:                  channelsWithWeights,
		strategy:                  strategy,
		autoDisableClosedChannels: autoDisableClosedChannels,
	}
	return ch, nil
}

type compositeChannelByPrioritization[T any, W any] struct {
	channelName               string
	channels                  []selectable.ChannelWithWeight[T, W]
	strategy                  PrioritizationStrategy[W]
	autoDisableClosedChannels bool
}

func (c *compositeChannelByPrioritization[T, W]) ChannelName() string {
	return c.channelName
}

func (c *compositeChannelByPrioritization[T, W]) NextSelectCases(upto int) ([]selectable.SelectCase[T], bool, *selectable.ClosedChannelDetails) {
	added := 0
	var selectCases []selectable.SelectCase[T]
	areAllCasesAdded := false
	nextSelectCasesIndexes := c.strategy.NextSelectCasesIndexes(upto)
	if len(nextSelectCasesIndexes) == 0 && upto == len(c.channels) {
		return nil, true, nil
	}

	for i, channelIndex := range nextSelectCasesIndexes {
		channelSelectCases, allSelected, closedChannel := c.channels[channelIndex].NextSelectCases(upto - added)
		if closedChannel != nil {
			return nil, true, &selectable.ClosedChannelDetails{
				ChannelName: closedChannel.ChannelName,
				PathInTree: append(closedChannel.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: channelIndex,
				}),
			}
		}
		if len(channelSelectCases) == 0 && (i == len(c.channels)-1) {
			return selectCases, true, nil
		}
		for _, sc := range channelSelectCases {
			selectCases = append(selectCases, selectable.SelectCase[T]{
				ChannelName: sc.ChannelName,
				MsgsC:       sc.MsgsC,
				PathInTree: append(sc.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: channelIndex,
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

func (c *compositeChannelByPrioritization[T, W]) UpdateOnCaseSelected(pathInTree []selectable.ChannelNode, recvOK bool) {
	if len(pathInTree) == 0 {
		return
	}
	if len(pathInTree) == 1 && !recvOK && c.autoDisableClosedChannels {
		c.strategy.DisableSelectCase(pathInTree[0].ChannelIndex)
		return
	}
	selectedChannel := c.channels[pathInTree[len(pathInTree)-1].ChannelIndex]
	selectedChannel.UpdateOnCaseSelected(pathInTree[:len(pathInTree)-1], recvOK)

	if recvOK {
		c.strategy.UpdateOnCaseSelected(pathInTree[len(pathInTree)-1].ChannelIndex)
	}
}
