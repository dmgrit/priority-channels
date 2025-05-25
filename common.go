package priority_channels

import (
	"context"
	"time"
)

type ReceiveStatus int

const (
	ReceiveStatusUnknown ReceiveStatus = iota
	ReceiveSuccess
	ReceiveContextCanceled
	ReceiveDefaultCase
	ReceiveChannelClosed
	ReceivePriorityChannelClosed
	ReceiveNoOpenChannels
)

func (r ReceiveStatus) ExitReason() ExitReason {
	switch r {
	case ReceiveChannelClosed:
		return ChannelClosed
	case ReceivePriorityChannelClosed:
		return PriorityChannelClosed
	case ReceiveNoOpenChannels:
		return NoOpenChannels
	case ReceiveContextCanceled:
		return ContextCanceled
	default:
		return UnknownExitReason
	}
}

type ExitReason int

const (
	UnknownExitReason ExitReason = iota
	ChannelClosed
	PriorityChannelClosed
	NoOpenChannels
	ContextCanceled
)

type PriorityChannelOptions struct {
	channelReceiveWaitInterval *time.Duration
	autoDisableClosedChannels  bool
	frequencyMode              *FrequencyMode
	frequencyMethod            *FrequencyMethod
}

const defaultChannelReceiveWaitInterval = 100 * time.Microsecond

func ChannelWaitInterval(d time.Duration) func(opt *PriorityChannelOptions) {
	return func(opt *PriorityChannelOptions) {
		opt.channelReceiveWaitInterval = &d
	}
}

func AutoDisableClosedChannels() func(opt *PriorityChannelOptions) {
	return func(opt *PriorityChannelOptions) {
		opt.autoDisableClosedChannels = true
	}
}

func WithFrequencyMethod(method FrequencyMethod) func(opt *PriorityChannelOptions) {
	return func(opt *PriorityChannelOptions) {
		opt.frequencyMethod = &method
	}
}

func WithFrequencyMode(mode FrequencyMode) func(opt *PriorityChannelOptions) {
	return func(opt *PriorityChannelOptions) {
		opt.frequencyMode = &mode
	}
}

func ProcessPriorityChannelMessages[T any](
	msgReceiver *PriorityChannel[T],
	msgProcessor func(ctx context.Context, msg T, channelName string),
	done chan<- ExitReason) {
	for {
		// There is no context per-message, but there is a single context for the entire priority-channel
		// On receiving the message we do not pass any specific context,
		// but on processing the message we pass the priority-channel context
		msg, channelName, status := msgReceiver.ReceiveWithContext(context.Background())
		if status != ReceiveSuccess {
			done <- status.ExitReason()
			return
		}
		msgProcessor(msgReceiver.ctx, msg, channelName)
	}
}

func getZero[T any]() T {
	var result T
	return result
}
