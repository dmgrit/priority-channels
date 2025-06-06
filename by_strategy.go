package priority_channels

import (
	"context"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/internal/selectable"
	"github.com/dmgrit/priority-channels/strategies"
)

func NewByStrategy[T any, W any](ctx context.Context,
	strategy strategies.PrioritizationStrategy[W],
	channelsWithWeights []channels.ChannelWithWeight[T, W],
	options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	selectableChannels := make([]selectable.ChannelWithWeight[T, W], 0, len(channelsWithWeights))
	for _, c := range channelsWithWeights {
		selectableChannels = append(selectableChannels, selectable.NewChannelWithWeight(c))
	}
	return newByStrategy(ctx, strategy, selectableChannels, options...)
}

func newByStrategy[T any, W any](ctx context.Context,
	strategy strategies.PrioritizationStrategy[W],
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
	channelNameToChannel := make(map[string]<-chan T)
	innerPriorityChannelsContexts := make(map[string]context.Context)
	if err := compositeChannel.GetInputAndInnerPriorityChannels(channelNameToChannel, innerPriorityChannelsContexts); err != nil {
		return nil, err
	}
	return newPriorityChannel(ctx, compositeChannel, channelNameToChannel, options...), nil
}

func newCompositeChannelByStrategy[T any, W any](name string,
	strategy strategies.PrioritizationStrategy[W],
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
	strategy                  strategies.PrioritizationStrategy[W]
	autoDisableClosedChannels bool
}

func (c *compositeChannelByPrioritization[T, W]) ChannelName() string {
	return c.channelName
}

func (c *compositeChannelByPrioritization[T, W]) NextSelectCases(upto int) ([]selectable.SelectCase[T], bool, *selectable.ClosedChannelDetails) {
	var selectCases []selectable.SelectCase[T]
	nextSelectCasesIndexes, areAllDirectChannelsSelected := c.strategy.NextSelectCasesRankedIndexes(upto)
	areAllCasesAddedSoFar := areAllDirectChannelsSelected

	prevChannelRank := -1
	totalAdded, maxCurrRankAdded := 0, 0

	for i, channelRankedIndex := range nextSelectCasesIndexes {
		currChannelIndex := channelRankedIndex.Index
		currChannelRank := channelRankedIndex.Rank
		currChannelAdded := 0
		if prevChannelRank != currChannelRank && i > 0 {
			totalAdded += maxCurrRankAdded
			if totalAdded >= upto {
				return selectCases, false, nil
			}
			maxCurrRankAdded = 0
		}
		prevChannelRank = currChannelRank
		currChannelSelectCases, allCurrDescendantsSelected, closedChannel := c.channels[currChannelIndex].NextSelectCases(upto - totalAdded)
		if closedChannel != nil {
			return nil, true, &selectable.ClosedChannelDetails{
				ChannelName: closedChannel.ChannelName,
				PathInTree: append(closedChannel.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: currChannelIndex,
				}),
			}
		}
		for j, sc := range currChannelSelectCases {
			selectCases = append(selectCases, selectable.SelectCase[T]{
				ChannelName: sc.ChannelName,
				MsgsC:       sc.MsgsC,
				PathInTree: append(sc.PathInTree, selectable.ChannelNode{
					ChannelName:  c.channelName,
					ChannelIndex: currChannelIndex,
				}),
			})
			currChannelAdded++
			if currChannelAdded > maxCurrRankAdded {
				maxCurrRankAdded = currChannelAdded
			}
			if totalAdded+currChannelAdded >= upto {
				areAllCasesAdded := areAllCasesAddedSoFar &&
					areAllDirectChannelsSelected &&
					(i == len(nextSelectCasesIndexes)-1) &&
					allCurrDescendantsSelected &&
					(j == len(currChannelSelectCases)-1)
				if areAllCasesAdded {
					return selectCases, areAllCasesAdded, nil
				}
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

func (c *compositeChannelByPrioritization[T, W]) RecoverClosedInputChannel(ch <-chan T, pathInTree []selectable.ChannelNode) {
	recoverFn := func(selectedChannel selectable.ChannelWithWeight[T, W], pathInTree []selectable.ChannelNode) {
		selectedChannel.RecoverClosedInputChannel(ch, pathInTree)
	}
	c.doRecoverClosedChannel(recoverFn, pathInTree)
}

func (c *compositeChannelByPrioritization[T, W]) RecoverClosedInnerPriorityChannel(ctx context.Context, pathInTree []selectable.ChannelNode) {
	recoverFn := func(selectedChannel selectable.ChannelWithWeight[T, W], pathInTree []selectable.ChannelNode) {
		selectedChannel.RecoverClosedInnerPriorityChannel(ctx, pathInTree)
	}
	c.doRecoverClosedChannel(recoverFn, pathInTree)
}

func (c *compositeChannelByPrioritization[T, W]) doRecoverClosedChannel(recoverFn func(selectable.ChannelWithWeight[T, W], []selectable.ChannelNode), pathInTree []selectable.ChannelNode) {
	if len(pathInTree) == 0 {
		return
	}
	if len(pathInTree) == 1 {
		selectedChannel := c.channels[pathInTree[0].ChannelIndex]
		recoverFn(selectedChannel, nil)
		c.strategy.EnableSelectCase(pathInTree[0].ChannelIndex)
		return
	}
	selectedChannel := c.channels[pathInTree[len(pathInTree)-1].ChannelIndex]
	recoverFn(selectedChannel, pathInTree[:len(pathInTree)-1])
}

func (c *compositeChannelByPrioritization[T, W]) GetInputAndInnerPriorityChannels(inputChannels map[string]<-chan T, innerPriorityChannels map[string]context.Context) error {
	for _, ch := range c.channels {
		if err := ch.GetInputAndInnerPriorityChannels(inputChannels, innerPriorityChannels); err != nil {
			return err
		}
	}
	return nil
}

func (c *compositeChannelByPrioritization[T, W]) GetInputChannelsPaths(m map[string][]selectable.ChannelNode, currPathInTree []selectable.ChannelNode) {
	for i, ch := range c.channels {
		ch.GetInputChannelsPaths(m, append(currPathInTree, selectable.ChannelNode{
			ChannelName:  ch.ChannelName(),
			ChannelIndex: i,
		}))
	}
}

func (c *compositeChannelByPrioritization[T, W]) Clone() selectable.Channel[T] {
	res := &compositeChannelByPrioritization[T, W]{
		channelName:               c.channelName,
		autoDisableClosedChannels: c.autoDisableClosedChannels,
		channels:                  make([]selectable.ChannelWithWeight[T, W], 0, len(c.channels)),
	}
	var weights []W
	for _, ch := range c.channels {
		res.channels = append(res.channels, ch.CloneChannelWithWeight())
		weights = append(weights, ch.Weight())
	}
	res.strategy = c.strategy.InitializeCopy()
	return res
}
