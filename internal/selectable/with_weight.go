package selectable

import (
	"context"
	"fmt"
	"github.com/dmgrit/priority-channels/channels"
)

type ChannelWithWeight[T any, W any] interface {
	Channel[T]
	Weight() W
	CloneChannelWithWeight() ChannelWithWeight[T, W]
}

type channelWithWeight[T any, W any] struct {
	channelName string
	msgsC       <-chan T
	weight      W
}

func (c *channelWithWeight[T, W]) ChannelName() string {
	return c.channelName
}

func (c *channelWithWeight[T, W]) NextSelectCases(upto int) ([]SelectCase[T], bool, *ClosedChannelDetails) {
	return []SelectCase[T]{
		{
			ChannelName: c.channelName,
			MsgsC:       c.msgsC,
		},
	}, true, nil
}

func (c *channelWithWeight[T, W]) UpdateOnCaseSelected(pathInTree []ChannelNode, recvOK bool) {}

func (c *channelWithWeight[T, W]) RecoverClosedInputChannel(ch <-chan T, pathInTree []ChannelNode) {
	c.msgsC = ch
}

func (c *channelWithWeight[T, W]) RecoverClosedInnerPriorityChannel(ctx context.Context, pathInTree []ChannelNode) {
	// this should never be called
}

func (c *channelWithWeight[T, W]) GetInputAndInnerPriorityChannels(inputChannels map[string]<-chan T, innerPriorityChannels map[string]context.Context) error {
	if _, ok := inputChannels[c.channelName]; ok {
		return fmt.Errorf("channel name '%s' is used more than once", c.channelName)
	}
	inputChannels[c.channelName] = c.msgsC
	return nil
}

func (c *channelWithWeight[T, W]) GetInputChannelsPaths(m map[string][]ChannelNode, currPathInTree []ChannelNode) {
	if _, ok := m[c.channelName]; !ok {
		return
	}
	m[c.channelName] = currPathInTree
}

func (c *channelWithWeight[T, W]) Clone() Channel[T] {
	return c.CloneChannelWithWeight()
}

func (c *channelWithWeight[T, W]) CloneChannelWithWeight() ChannelWithWeight[T, W] {
	return &channelWithWeight[T, W]{
		channelName: c.channelName,
		msgsC:       c.msgsC,
		weight:      c.weight,
	}
}

func (c *channelWithWeight[T, W]) Weight() W {
	return c.weight
}

func NewChannelWithWeight[T any, W any](ch channels.ChannelWithWeight[T, W]) ChannelWithWeight[T, W] {
	return &channelWithWeight[T, W]{
		channelName: ch.ChannelName(),
		msgsC:       ch.MsgsC(),
		weight:      ch.Weight(),
	}
}
