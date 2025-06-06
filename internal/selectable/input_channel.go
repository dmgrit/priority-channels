package selectable

import (
	"context"
	"fmt"
)

type inputChannel[T any] struct {
	channelName string
	msgsC       <-chan T
}

func (c *inputChannel[T]) ChannelName() string {
	return c.channelName
}

func (c *inputChannel[T]) NextSelectCases(upto int) ([]SelectCase[T], bool, *ClosedChannelDetails) {
	return []SelectCase[T]{
		{
			ChannelName: c.channelName,
			MsgsC:       c.msgsC,
		},
	}, true, nil
}

func (c *inputChannel[T]) UpdateOnCaseSelected(pathInTree []ChannelNode, recvOK bool) {}

func (c *inputChannel[T]) RecoverClosedInputChannel(ch <-chan T, pathInTree []ChannelNode) {
	c.msgsC = ch
}

func (c *inputChannel[T]) RecoverClosedInnerPriorityChannel(ctx context.Context, pathInTree []ChannelNode) {
	// this should never be called
}

func (c *inputChannel[T]) GetInputAndInnerPriorityChannels(inputChannels map[string]<-chan T, innerPriorityChannels map[string]context.Context) error {
	if _, ok := inputChannels[c.channelName]; ok {
		return fmt.Errorf("channel name '%s' is used more than once", c.channelName)
	}
	inputChannels[c.channelName] = c.msgsC
	return nil
}

func (c *inputChannel[T]) GetInputChannelsPaths(m map[string][]ChannelNode, currPathInTree []ChannelNode) {
	if _, ok := m[c.channelName]; !ok {
		return
	}
	m[c.channelName] = currPathInTree
}

func (c *inputChannel[T]) Clone() Channel[T] {
	return &inputChannel[T]{
		channelName: c.channelName,
		msgsC:       c.msgsC,
	}
}

func NewFromInputChannel[T any](channelName string, msgsC <-chan T) Channel[T] {
	return &inputChannel[T]{
		channelName: channelName,
		msgsC:       msgsC,
	}
}
