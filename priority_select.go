package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/channels"
)

func Select[T any](ctx context.Context,
	channelsWithPriorities []channels.ChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (msg T, channelName string, status ReceiveStatus, err error) {
	pc, err := NewByHighestAlwaysFirst(context.Background(), channelsWithPriorities, options...)
	if err != nil {
		return getZero[T](), "", ReceiveStatusUnknown, err
	}
	msg, channelName, status = pc.ReceiveWithContext(ctx)
	return
}

func SelectWithDefaultCase[T any](
	channelsWithPriorities []channels.ChannelWithPriority[T],
	options ...func(*PriorityChannelOptions)) (msg T, channelName string, status ReceiveStatus, err error) {
	pc, err := NewByHighestAlwaysFirst(context.Background(), channelsWithPriorities, options...)
	if err != nil {
		return getZero[T](), "", ReceiveStatusUnknown, err
	}
	msg, channelName, status = pc.ReceiveWithDefaultCase()
	return
}
