package selectable

import (
	"github.com/dmgrit/priority-channels/channels"
)

type ChannelWithWeight[T any, W any] interface {
	Channel[T]
	Weight() W
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

func (c *channelWithWeight[T, W]) RecoverClosedChannel(ch <-chan T, pathInTree []ChannelNode) {
	c.msgsC = ch
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
