package priority_channels

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type PriorityConsumer[T any] struct {
	ctx                       context.Context
	channelNameToChannel      map[string]<-chan T
	priorityChannel           *PriorityChannel[T]
	priorityChannelConfig     Configuration
	priorityChannelUpdatesMtx sync.Mutex
	priorityChannelUpdatesC   chan *PriorityChannel[T]
	priorityChannelClosedC    chan struct{}
	isClosing                 bool
	isClosed                  bool
	exitReason                ExitReason
	exitReasonChannelName     string
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
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.priorityChannelUpdatesC != nil {
		return nil, errors.New("consume already called")
	} else if c.isClosing {
		return nil, errors.New("cannot consume after closing")
	}

	deliveries := make(chan T)
	c.priorityChannelUpdatesC = make(chan *PriorityChannel[T], 1)
	go func() {
		defer close(deliveries)
		for {
			select {
			case priorityChannel, ok := <-c.priorityChannelUpdatesC:
				c.priorityChannel.Close()
				if !ok {
					c.setClosed(PriorityChannelClosed, "")
					return
				}
				c.priorityChannel = priorityChannel
			default:
				// There is no context per-message, but there is a single context for the entire priority-channel
				// On receiving the message we do not pass any specific context,
				// but on processing the message we pass the priority-channel context
				msg, channelName, status := c.priorityChannel.ReceiveWithContext(context.Background())
				if status != ReceiveSuccess {
					c.setClosed(status.ExitReason(), channelName)
					return
				}
				deliveries <- msg
			}
		}
	}()

	return deliveries, nil
}

func (c *PriorityConsumer[T]) UpdatePriorityConfiguration(priorityConfiguration Configuration) error {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.priorityChannelUpdatesC == nil {
		return errors.New("cannot update priority channel configuration before consuming has started")
	}
	if c.isClosing {
		return errors.New("cannot update priority channel configuration after closing")
	}

	priorityChannel, err := NewFromConfiguration(c.ctx, priorityConfiguration, c.channelNameToChannel)
	if err != nil {
		return fmt.Errorf("failed to create priority channel from configuration: %w", err)
	}
	if priorityChannel == nil {
		return errors.New("failed to create priority channel from configuration")
	}

	select {
	case c.priorityChannelUpdatesC <- priorityChannel:
	default:
		return errors.New("priority configuration update is already in progress, please retry later")
	}

	return nil
}

func (c *PriorityConsumer[T]) Close(wait bool) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if !c.isClosing {
		c.isClosing = true
		close(c.priorityChannelUpdatesC)
	}

	if wait {
		<-c.priorityChannelClosedC
	}
}

func (c *PriorityConsumer[T]) IsClosed() (bool, ExitReason, string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.isClosed {
		return true, c.exitReason, c.exitReasonChannelName
	}
	return false, UnknownExitReason, ""
}

func (c *PriorityConsumer[T]) setClosed(exitReason ExitReason, exitReasonChannelName string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	c.isClosed = true
	c.exitReason = exitReason
	c.exitReasonChannelName = exitReasonChannelName
	close(c.priorityChannelClosedC)
}
