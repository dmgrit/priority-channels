package selectable

import (
	"errors"

	"github.com/dmgrit/priority-channels/channels"
)

type ChannelWithPriority[T any] interface {
	Channel[T]
	Priority() int
}

type channelWithPriority[T any] struct {
	channelName string
	msgsC       <-chan T
	priority    int
}

func (c *channelWithPriority[T]) ChannelName() string {
	return c.channelName
}

func (c *channelWithPriority[T]) NextSelectCases(upto int) ([]SelectCase[T], bool, *ClosedChannelDetails) {
	return []SelectCase[T]{
		{
			ChannelName: c.channelName,
			MsgsC:       c.msgsC,
		},
	}, true, nil
}

func (c *channelWithPriority[T]) UpdateOnCaseSelected(pathInTree []ChannelNode) {}

func (c *channelWithPriority[T]) Priority() int {
	return c.priority
}

func (c *channelWithPriority[T]) Validate() error {
	if c.priority < 0 {
		return errors.New("priority cannot be negative")
	}
	return nil
}

func NewChannelWithPriority[T any](ch channels.ChannelWithPriority[T]) ChannelWithPriority[T] {
	return &channelWithPriority[T]{
		channelName: ch.ChannelName(),
		msgsC:       ch.MsgsC(),
		priority:    ch.Priority(),
	}
}
