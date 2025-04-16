package priority_channels

import (
	"context"
	"errors"
	"fmt"
)

type PriorityConsumer[T any] struct {
	ctx                     context.Context
	channelNameToChannel    map[string]<-chan T
	priorityChannel         *PriorityChannel[T]
	priorityChannelConfig   Configuration
	priorityChannelUpdatesC chan *PriorityChannel[T]
	priorityChannelClosedC  chan struct{}
	exitReason              ExitReason
	exitReasonChannelName   string
}

func NewConsumer[T any](
	ctx context.Context,
	channelNameToChannel map[string]<-chan T,
	priorityConfiguration Configuration,
) (*PriorityConsumer[T], error) {
	priorityChannel, err := NewFromConfiguration(ctx, priorityConfiguration, channelNameToChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to create priority channel from configuration: %w", err)
	}
	return &PriorityConsumer[T]{
		ctx:                    ctx,
		priorityChannel:        priorityChannel,
		channelNameToChannel:   channelNameToChannel,
		priorityChannelConfig:  priorityConfiguration,
		priorityChannelClosedC: make(chan struct{}),
	}, nil
}

func (c *PriorityConsumer[T]) Consume() (<-chan T, error) {
	deliveries := make(chan T)
	c.priorityChannelUpdatesC = make(chan *PriorityChannel[T])
	go func() {
		defer close(deliveries)
		defer close(c.priorityChannelClosedC)
		for {
			select {
			case priorityChannel := <-c.priorityChannelUpdatesC:
				c.priorityChannel.Close()
				if priorityChannel == nil {
					c.exitReason = PriorityChannelClosed
					return
				}
				c.priorityChannel = priorityChannel
			default:
				// There is no context per-message, but there is a single context for the entire priority-channel
				// On receiving the message we do not pass any specific context,
				// but on processing the message we pass the priority-channel context
				msg, channelName, status := c.priorityChannel.ReceiveWithContext(context.Background())
				if status != ReceiveSuccess {
					c.exitReason = status.ExitReason()
					c.exitReasonChannelName = channelName
					return
				}
				deliveries <- msg
			}
		}
	}()

	return deliveries, nil
}

func (c *PriorityConsumer[T]) UpdatePriorityConfiguration(priorityConfiguration Configuration) error {
	if c.priorityChannelUpdatesC == nil {
		return errors.New("cannot update priority channel configuration before consuming has started")
	}
	priorityChannel, err := NewFromConfiguration(c.ctx, priorityConfiguration, c.channelNameToChannel)
	if err != nil {
		return fmt.Errorf("failed to create priority channel from configuration: %w", err)
	}
	if priorityChannel == nil {
		return errors.New("failed to create priority channel from configuration")
	}
	go func() {
		c.priorityChannelUpdatesC <- priorityChannel
	}()
	return nil
}

func (c *PriorityConsumer[T]) Close() {
	if c.priorityChannelUpdatesC != nil {
		c.priorityChannelUpdatesC <- nil
	}
}

func (c *PriorityConsumer[T]) IsClosed() (bool, ExitReason, string) {
	select {
	case <-c.priorityChannelClosedC:
		return true, c.exitReason, c.exitReasonChannelName
	default:
		return false, UnknownExitReason, ""
	}
}
