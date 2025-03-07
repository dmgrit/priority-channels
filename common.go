package priority_channels

import (
	"context"
	"time"
)

type ReceiveStatus int

const (
	ReceiveSuccess ReceiveStatus = iota
	ReceiveContextCancelled
	ReceiveDefaultCase
	ReceiveChannelClosed
	ReceivePriorityChannelClosed
	ReceiveNoOpenChannels
	ReceiveStatusUnknown
)

func (r ReceiveStatus) ExitReason() ExitReason {
	switch r {
	case ReceiveChannelClosed:
		return ChannelClosed
	case ReceivePriorityChannelClosed:
		return PriorityChannelClosed
	case ReceiveNoOpenChannels:
		return NoOpenChannels
	default:
		return UnknownExitReason
	}
}

type ExitReason int

const (
	ChannelClosed ExitReason = iota
	PriorityChannelClosed
	NoOpenChannels
	UnknownExitReason
)

type FrequencyMethod int

const (
	StrictOrderAcrossCycles FrequencyMethod = iota
	StrictOrderFully
	ProbabilisticByCaseDuplication
	ProbabilisticByMultipleRandCalls
)

type PriorityChannelOptions struct {
	channelReceiveWaitInterval *time.Duration
	autoDisableClosedChannels  bool
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
