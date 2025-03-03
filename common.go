package priority_channels

import (
	"context"
	"time"
)

type ReceiveStatus int

const (
	ReceiveSuccess ReceiveStatus = iota
	ReceiveContextCancelled
	ReceiveChannelClosed
	ReceiveDefaultCase
	ReceivePriorityChannelCancelled
	ReceiveNoOpenChannels
	ReceiveStatusUnknown
)

func (r ReceiveStatus) ExitReason() ExitReason {
	switch r {
	case ReceiveChannelClosed:
		return ChannelClosed
	case ReceivePriorityChannelCancelled:
		return PriorityChannelCancelled
	default:
		return UnknownExitReason
	}
}

type ExitReason int

const (
	ChannelClosed ExitReason = iota
	PriorityChannelCancelled
	UnknownExitReason
)

type PriorityChannelOptions struct {
	channelReceiveWaitInterval *time.Duration
	autoDisableClosedChannels  bool
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

func ProcessPriorityChannelMessages[T any](
	msgReceiver *PriorityChannel[T],
	msgProcessor func(ctx context.Context, msg T, channelName string)) ExitReason {
	for {
		// There is no context per-message, but there is a single context for the entire priority-channel
		// On receiving the message we do not pass any specific context,
		// but on processing the message we pass the priority-channel context
		msg, channelName, status := msgReceiver.ReceiveWithContext(context.Background())
		if status != ReceiveSuccess {
			return status.ExitReason()
		}
		msgProcessor(msgReceiver.ctx, msg, channelName)
	}
}

func getZero[T any]() T {
	var result T
	return result
}
