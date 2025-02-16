package priority_channels

import (
	"context"
	"time"

	"github.com/dmgrit/priority-channels/channels"
)

type PriorityChannel[T any] interface {
	Receive() (msg T, channelName string, ok bool)
	ReceiveWithContext(ctx context.Context) (msg T, channelName string, status ReceiveStatus)
	ReceiveWithDefaultCase() (msg T, channelName string, status ReceiveStatus)
	AsSelectableChannelWithPriority(name string, priority int) channels.ChannelWithPriority[T]
	AsSelectableChannelWithFreqRatio(name string, freqRatio int) channels.ChannelWithFreqRatio[T]
}

type PriorityChannelWithContext[T any] interface {
	PriorityChannel[T]
	Context() context.Context
}

type ReceiveStatus int

const (
	ReceiveSuccess ReceiveStatus = iota
	ReceiveContextCancelled
	ReceiveChannelClosed
	ReceiveDefaultCase
	ReceivePriorityChannelCancelled
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
}

const defaultChannelReceiveWaitInterval = 100 * time.Microsecond

func ChannelWaitInterval(d time.Duration) func(opt *PriorityChannelOptions) {
	return func(opt *PriorityChannelOptions) {
		opt.channelReceiveWaitInterval = &d
	}
}

func ProcessPriorityChannelMessages[T any](
	msgReceiver PriorityChannel[T],
	msgProcessor func(ctx context.Context, msg T, channelName string)) ExitReason {
	for {
		// There is no context per-message, but there is a single context for the entire priority-channel
		// On receiving the message we do not pass any specific context,
		// but on processing the message we pass the priority-channel context
		msg, channelName, status := msgReceiver.ReceiveWithContext(context.Background())
		if status != ReceiveSuccess {
			return status.ExitReason()
		}
		ctx := context.Background()
		msgReceiverWithContext, ok := msgReceiver.(PriorityChannelWithContext[T])
		if ok {
			ctx = msgReceiverWithContext.Context()
		}
		msgProcessor(ctx, msg, channelName)
	}
}

func getZero[T any]() T {
	var result T
	return result
}
