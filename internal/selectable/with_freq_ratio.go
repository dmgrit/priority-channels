package selectable

import (
	"errors"

	"github.com/dmgrit/priority-channels/channels"
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

func (c *channelWithFreqRatio[T]) NextSelectCases(upto int) ([]SelectCase[T], bool, *ClosedChannelDetails) {
	return []SelectCase[T]{
		{
			ChannelName: c.channelName,
			MsgsC:       c.msgsC,
		},
	}, true, nil
}

func (c *channelWithFreqRatio[T]) UpdateOnCaseSelected(pathInTree []ChannelNode) {}

func (c *channelWithFreqRatio[T]) FreqRatio() int {
	return c.freqRatio
}

func (c *channelWithFreqRatio[T]) Validate() error {
	if c.freqRatio <= 0 {
		return errors.New("frequency ratio must be greater than 0")
	}
	return nil
}

func NewChannelWithFreqRatio[T any](ch channels.ChannelWithFreqRatio[T]) ChannelWithFreqRatio[T] {
	return &channelWithFreqRatio[T]{
		channelName: ch.ChannelName(),
		msgsC:       ch.MsgsC(),
		freqRatio:   ch.FreqRatio(),
	}
}
