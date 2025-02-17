package channels

import (
	"errors"
)

var ErrPriorityIsNegative = errors.New("priority cannot be negative")

type ChannelWithPriority[T any] struct {
	channelName string
	msgsC       <-chan T
	priority    int
}

func (c *ChannelWithPriority[T]) ChannelName() string {
	return c.channelName
}

func (c *ChannelWithPriority[T]) MsgsC() <-chan T {
	return c.msgsC
}

func (c *ChannelWithPriority[T]) Priority() int {
	return c.priority
}

func (c *ChannelWithPriority[T]) Validate() error {
	if c.priority < 0 {
		return ErrPriorityIsNegative
	}
	return nil
}

func NewChannelWithPriority[T any](channelName string, msgsC <-chan T, priority int) ChannelWithPriority[T] {
	return ChannelWithPriority[T]{
		channelName: channelName,
		msgsC:       msgsC,
		priority:    priority,
	}
}
