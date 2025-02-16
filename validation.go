package priority_channels

import (
	"errors"
	"fmt"
	"github.com/dmgrit/priority-channels/channels"
)

var (
	ErrNoChannels       = errors.New("no channels provided")
	ErrEmptyChannelName = errors.New("channel name is empty")
)

type DuplicateChannelError struct {
	ChannelName string
}

func (e *DuplicateChannelError) Error() string {
	return fmt.Sprintf("channel name '%s' is used more than once", e.ChannelName)
}

type ChannelValidationError struct {
	ChannelName string
	Err         error
}

func (e *ChannelValidationError) Error() string {
	return fmt.Sprintf("channel '%s': %v", e.ChannelName, e.Err)
}

type FunctionNotSetError struct {
	FuncName string
}

func (e *FunctionNotSetError) Error() string {
	return fmt.Sprintf("%s  function is not set", e.FuncName)
}

func validateInputChannels[T any](channels []channels.SelectableChannel[T]) error {
	if len(channels) == 0 {
		return ErrNoChannels
	}
	channelNames := make(map[string]struct{})
	for _, c := range channels {
		if c.ChannelName() == "" {
			return ErrEmptyChannelName
		}
		if err := c.Validate(); err != nil {
			return &ChannelValidationError{ChannelName: c.ChannelName(), Err: err}
		}
		if _, ok := channelNames[c.ChannelName()]; ok {
			return &DuplicateChannelError{ChannelName: c.ChannelName()}
		}
		channelNames[c.ChannelName()] = struct{}{}
	}
	return nil
}

func convertChannelsWithPrioritiesToChannels[T any](channelsWithPriorities []channels.ChannelWithPriority[T]) []channels.SelectableChannel[T] {
	res := make([]channels.SelectableChannel[T], 0, len(channelsWithPriorities))
	for _, c := range channelsWithPriorities {
		res = append(res, c)
	}
	return res
}

func convertChannelsWithFreqRatiosToChannels[T any](channelsWithFreqRatios []channels.ChannelWithFreqRatio[T]) []channels.SelectableChannel[T] {
	res := make([]channels.SelectableChannel[T], 0, len(channelsWithFreqRatios))
	for _, c := range channelsWithFreqRatios {
		res = append(res, c)
	}
	return res
}
