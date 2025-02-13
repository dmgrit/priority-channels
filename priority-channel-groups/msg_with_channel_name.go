package priority_channel_groups

import (
	"context"
	"sync"

	"github.com/dmgrit/priority-channels"
)

type msgWithChannelName[T any] struct {
	Msg         T
	ChannelName string
}

type priorityChannelOfMsgsWithChannelName[T any] struct {
	priorityChannel priority_channels.PriorityChannel[msgWithChannelName[T]]
}

func (pc *priorityChannelOfMsgsWithChannelName[T]) Receive() (msg T, channelName string, ok bool) {
	msgWithChannelName, _, ok := pc.priorityChannel.Receive()
	if !ok {
		return getZero[T](), "", false
	}
	return msgWithChannelName.Msg, msgWithChannelName.ChannelName, true
}

func (pc *priorityChannelOfMsgsWithChannelName[T]) ReceiveWithContext(ctx context.Context) (msg T, channelName string, status priority_channels.ReceiveStatus) {
	msgWithChannelName, channelName, status := pc.priorityChannel.ReceiveWithContext(ctx)
	if status != priority_channels.ReceiveSuccess {
		return getZero[T](), channelName, status
	}
	return msgWithChannelName.Msg, msgWithChannelName.ChannelName, status
}

func (pc *priorityChannelOfMsgsWithChannelName[T]) ReceiveWithDefaultCase() (msg T, channelName string, status priority_channels.ReceiveStatus) {
	msgWithChannelName, channelName, status := pc.priorityChannel.ReceiveWithDefaultCase()
	if status != priority_channels.ReceiveSuccess {
		return getZero[T](), channelName, status
	}
	return msgWithChannelName.Msg, msgWithChannelName.ChannelName, status
}

func processPriorityChannelToMsgsWithChannelName[T any](ctx context.Context, priorityChannel priority_channels.PriorityChannel[T]) (
	msgWithNameC <-chan msgWithChannelName[T],
	fnGetClosedChannelDetails func() (string, priority_channels.ReceiveStatus),
	fnIsReady func() bool) {

	resC := make(chan msgWithChannelName[T])
	var closedChannelName string
	var closedChannelStatus priority_channels.ReceiveStatus
	var mtxClosedChannelDetails sync.RWMutex

	go func() {
		for {
			message, channelName, status := priorityChannel.ReceiveWithContext(ctx)
			if status == priority_channels.ReceiveContextCancelled {
				return
			}
			if status != priority_channels.ReceiveSuccess {
				mtxClosedChannelDetails.Lock()
				closedChannelName = channelName
				closedChannelStatus = status
				mtxClosedChannelDetails.Unlock()
				close(resC)
				return
			}
			select {
			case <-ctx.Done():
				return
			case resC <- msgWithChannelName[T]{Msg: message, ChannelName: channelName}:
			}
		}
	}()

	resFnGetClosedChannelDetails := func() (string, priority_channels.ReceiveStatus) {
		mtxClosedChannelDetails.RLock()
		defer mtxClosedChannelDetails.RUnlock()
		return closedChannelName, closedChannelStatus
	}
	var resFnIsReady func() bool
	readinessChecker, ok := priorityChannel.(priority_channels.ReadinessChecker)
	if ok {
		resFnIsReady = readinessChecker.IsReady
	} else {
		resFnIsReady = func() bool {
			return true
		}
	}
	return resC, resFnGetClosedChannelDetails, resFnIsReady
}

func getZero[T any]() T {
	var result T
	return result
}
