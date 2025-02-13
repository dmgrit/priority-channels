package priority_channel_groups

import (
	"context"
	"sort"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func CombineByHighestPriorityFirst[T any](ctx context.Context,
	priorityChannelsWithPriority []PriorityChannelWithPriority[T],
	options ...func(*priority_channels.PriorityChannelOptions)) (priority_channels.PriorityChannel[T], error) {
	channels := newPriorityChannelsGroupByHighestPriorityFirst[T](ctx, priorityChannelsWithPriority)
	priorityChannel, err := priority_channels.NewByHighestAlwaysFirst[msgWithChannelName[T]](ctx, channels, options...)
	if err != nil {
		return nil, err
	}
	return &priorityChannelOfMsgsWithChannelName[T]{priorityChannel: priorityChannel}, nil
}

type PriorityChannelWithPriority[T any] interface {
	Name() string
	PriorityChannel() priority_channels.PriorityChannel[T]
	Priority() int
}

type priorityChannelWithPriority[T any] struct {
	name            string
	priorityChannel priority_channels.PriorityChannel[T]
	priority        int
}

func (c *priorityChannelWithPriority[T]) Name() string {
	return c.name
}

func (c *priorityChannelWithPriority[T]) PriorityChannel() priority_channels.PriorityChannel[T] {
	return c.priorityChannel
}

func (c *priorityChannelWithPriority[T]) Priority() int {
	return c.priority
}

func NewPriorityChannelWithPriority[T any](name string, priorityChannel priority_channels.PriorityChannel[T], priority int) PriorityChannelWithPriority[T] {
	return &priorityChannelWithPriority[T]{
		name:            name,
		priorityChannel: priorityChannel,
		priority:        priority,
	}
}

func newPriorityChannelsGroupByHighestPriorityFirst[T any](
	ctx context.Context,
	priorityChannelsWithPriority []PriorityChannelWithPriority[T]) []channels.ChannelWithPriority[msgWithChannelName[T]] {
	res := make([]channels.ChannelWithPriority[msgWithChannelName[T]], 0, len(priorityChannelsWithPriority))

	for _, q := range priorityChannelsWithPriority {
		msgWithNameC, fnGetClosedChannelDetails, fnIsReady := processPriorityChannelToMsgsWithChannelName(ctx, q.PriorityChannel())
		channel := channels.NewChannelWithPriority[msgWithChannelName[T]](q.Name(), msgWithNameC, q.Priority())
		res = append(res, newChannelWithPriorityAndClosedChannelDetails(channel, fnGetClosedChannelDetails, fnIsReady))
	}
	sort.Slice(res, func(i int, j int) bool {
		return res[i].Priority() > res[j].Priority()
	})
	return res
}

type channelWithPriorityAndClosedChannelDetails[T any] struct {
	channel                   channels.ChannelWithPriority[T]
	fnGetClosedChannelDetails func() (string, priority_channels.ReceiveStatus)
	fnIsReady                 func() bool
}

func (c *channelWithPriorityAndClosedChannelDetails[T]) ChannelName() string {
	return c.channel.ChannelName()
}

func (c *channelWithPriorityAndClosedChannelDetails[T]) MsgsC() <-chan T {
	return c.channel.MsgsC()
}

func (c *channelWithPriorityAndClosedChannelDetails[T]) Priority() int {
	return c.channel.Priority()
}

func (c *channelWithPriorityAndClosedChannelDetails[T]) GetUnderlyingClosedChannelDetails() (string, priority_channels.ReceiveStatus) {
	return c.fnGetClosedChannelDetails()
}

func (c *channelWithPriorityAndClosedChannelDetails[T]) IsReady() bool {
	return c.fnIsReady()
}

func (c *channelWithPriorityAndClosedChannelDetails[T]) Validate() error {
	if err := c.channel.Validate(); err != nil {
		return err
	}
	if c.fnGetClosedChannelDetails == nil {
		return &priority_channels.FunctionNotSetError{FuncName: "GetUnderlyingClosedChannelDetails"}
	}
	if c.fnIsReady == nil {
		return &priority_channels.FunctionNotSetError{FuncName: "IsReady"}
	}
	return nil
}

func newChannelWithPriorityAndClosedChannelDetails[T any](
	channel channels.ChannelWithPriority[T],
	fnGetClosedChannelDetails func() (string, priority_channels.ReceiveStatus),
	fnIsReady func() bool) channels.ChannelWithPriority[T] {
	return &channelWithPriorityAndClosedChannelDetails[T]{
		channel:                   channel,
		fnGetClosedChannelDetails: fnGetClosedChannelDetails,
		fnIsReady:                 fnIsReady,
	}
}
