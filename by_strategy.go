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
	NextSelectCasesIndexes(upto int) ([]int, bool)
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
	nextSelectCasesIndexes, areAllDirectChannelsSelected := c.strategy.NextSelectCasesIndexes(upto)
	areAllCasesAddedSoFar := areAllDirectChannelsSelected

	for i, channelIndex := range nextSelectCasesIndexes {
		currChannelSelectCases, allCurrDescendantsSelected, closedChannel := c.channels[channelIndex].NextSelectCases(upto - added)
		if closedChannel != nil {
			return nil, true, &selectable.ClosedChannelDetails{
				ChannelName: closedChannel.ChannelName,
				PathInTree: append(closedChannel.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: channelIndex,
				}),
			}
		}
		for j, sc := range currChannelSelectCases {
			selectCases = append(selectCases, selectable.SelectCase[T]{
				ChannelName: sc.ChannelName,
				MsgsC:       sc.MsgsC,
				PathInTree: append(sc.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: channelIndex,
				}),
			})
			added++
			if added == upto {
				areAllCasesAdded := areAllCasesAddedSoFar &&
					areAllDirectChannelsSelected &&
					(i == len(nextSelectCasesIndexes)-1) &&
					allCurrDescendantsSelected &&
					(j == len(currChannelSelectCases)-1)
				return selectCases, areAllCasesAdded, nil
			}
		}
		areAllCasesAddedSoFar = areAllCasesAddedSoFar && allCurrDescendantsSelected
	}
	return selectCases, areAllCasesAddedSoFar, nil
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
