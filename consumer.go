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
	forceShutdownChannel      chan struct{}
	onMessageDrop             func(msg T, channelName string)
	isStopping                bool
	isStopped                 bool
	exitReason                ExitReason
	exitReasonChannelName     string
}

type Delivery[T any] struct {
	Msg            T
	ReceiveDetails ReceiveDetails
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
		forceShutdownChannel:   make(chan struct{}),
	}, nil
}

func (c *PriorityConsumer[T]) Consume() (<-chan Delivery[T], error) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.priorityChannelUpdatesC != nil {
		return nil, errors.New("consume already called")
	} else if c.isStopping {
		return nil, errors.New("cannot consume after stopping")
	}

	deliveries := make(chan Delivery[T])
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
				msg, receiveDetails, status := c.priorityChannel.ReceiveWithContextEx(context.Background())
				channelName := receiveDetails.ChannelName
				if status != ReceiveSuccess {
					c.setClosed(status.ExitReason(), channelName)
					return
				}
				select {
				case deliveries <- Delivery[T]{Msg: msg, ReceiveDetails: receiveDetails}:
				case <-c.forceShutdownChannel:
					if c.onMessageDrop != nil {
						c.onMessageDrop(msg, channelName)
					}
					c.setClosed(PriorityChannelClosed, "")
					return
				}
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
	if c.isStopping {
		return errors.New("cannot update priority channel configuration after stopping")
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

// StopGracefully stops the consumer with a graceful shutdown, draining the unprocessed messages before stopping.
func (c *PriorityConsumer[T]) StopGracefully() {
	c.stop(true, nil)
}

// StopImmediately stops the consumer in a forced manner.
// onMessageDrop is called when a message is dropped. It is optional and can be nil, in this case the message will be silently dropped.
func (c *PriorityConsumer[T]) StopImmediately(onMessageDrop func(msg T, channelName string)) {
	c.stop(false, onMessageDrop)
}

// Done returns a channel that is closed when the consumer is stopped.
func (c *PriorityConsumer[T]) Done() <-chan struct{} {
	return c.priorityChannelClosedC
}

func (c *PriorityConsumer[T]) stop(graceful bool, onMessageDrop func(msg T, channelName string)) {
	c.priorityChannelUpdatesMtx.Lock()
	if !c.isStopping {
		c.isStopping = true
		c.onMessageDrop = onMessageDrop
		c.priorityChannel.Close()
		close(c.priorityChannelUpdatesC)
	}
	c.priorityChannelUpdatesMtx.Unlock()

	if !graceful {
		c.forceShutdownChannel <- struct{}{}
	}
	<-c.priorityChannelClosedC
}

// Status returns whether the consumer is stopped, and if so, the reason for stopping and,
// in case the reason is a closed channel, the name of the channel that was closed.
func (c *PriorityConsumer[T]) Status() (stopped bool, reason ExitReason, channelName string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.isStopped {
		return true, c.exitReason, c.exitReasonChannelName
	}
	return false, UnknownExitReason, ""
}

func (c *PriorityConsumer[T]) setClosed(exitReason ExitReason, exitReasonChannelName string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	c.isStopped = true
	c.exitReason = exitReason
	c.exitReasonChannelName = exitReasonChannelName
	close(c.priorityChannelClosedC)
}
