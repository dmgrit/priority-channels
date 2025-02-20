package priority_channels

import (
	"errors"
	"fmt"

	"github.com/dmgrit/priority-channels/internal/selectable"
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

func validateInputChannels[T any](channels []selectable.Channel[T]) error {
	if len(channels) == 0 {
		return ErrNoChannels
	}
	channelNames := make(map[string]struct{})
	for _, c := range channels {
		if c.ChannelName() == "" {
			return ErrEmptyChannelName
		}
		if _, ok := channelNames[c.ChannelName()]; ok {
			return &DuplicateChannelError{ChannelName: c.ChannelName()}
		}
		channelNames[c.ChannelName()] = struct{}{}
	}
	return nil
}

func convertChannelsWithWeightsToChannels[T any, W any](channelsWithWeights []selectable.ChannelWithWeight[T, W]) []selectable.Channel[T] {
	res := make([]selectable.Channel[T], 0, len(channelsWithWeights))
	for _, c := range channelsWithWeights {
		res = append(res, c)
	}
	return res
}
