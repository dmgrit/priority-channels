package channels

import (
	"errors"
)

type ChannelWithFreqRatio[T any] interface {
	Channel[T]
	FreqRatio() int
}

type channelWithFreqRatio[T any] struct {
	channelName string
	msgsC       <-chan T
	freqRatio   int
}

func (c *channelWithFreqRatio[T]) ChannelName() string {
	return c.channelName
}

func (c *channelWithFreqRatio[T]) MsgsC() <-chan T {
	return c.msgsC
}

func (c *channelWithFreqRatio[T]) FreqRatio() int {
	return c.freqRatio
}

func (c *channelWithFreqRatio[T]) Validate() error {
	if c.freqRatio <= 0 {
		return errors.New("frequency ratio must be greater than 0")
	}
	return nil
}

func NewChannelWithFreqRatio[T any](channelName string, msgsC <-chan T, freqRatio int) ChannelWithFreqRatio[T] {
	return &channelWithFreqRatio[T]{
		channelName: channelName,
		msgsC:       msgsC,
		freqRatio:   freqRatio,
	}
}
