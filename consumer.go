package priority_channels

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type PriorityConsumer[T any] struct {
	ctx                       context.Context
	priorityChannel           *PriorityChannel[T]
	priorityChannelUpdatesMtx sync.Mutex
	priorityChannelClosedC    chan struct{}
	forceShutdownChannel      chan struct{}
	onMessageDrop             func(msg T, channelName string)
	started                   bool
	isStopping                bool
	status                    ProcessingStatus
	exitReason                ExitReason
	exitReasonChannelName     string
	closureBehaviour          ClosureBehavior
}

type Delivery[T any] struct {
	Msg            T
	ReceiveDetails ReceiveDetails
}

func NewConsumer[T any](
	ctx context.Context,
	channelNameToChannel map[string]<-chan T,
	innerPriorityChannelsContexts map[string]context.Context,
	priorityConfiguration Configuration,
	closureBehaviour ClosureBehavior,
) (*PriorityConsumer[T], error) {
	priorityChannel, err := NewFromConfiguration(ctx, priorityConfiguration, channelNameToChannel, innerPriorityChannelsContexts)
	if err != nil {
		return nil, fmt.Errorf("failed to create priority channel from configuration: %w", err)
	}
	return &PriorityConsumer[T]{
		ctx:                    ctx,
		priorityChannel:        priorityChannel,
		priorityChannelClosedC: make(chan struct{}),
		forceShutdownChannel:   make(chan struct{}),
		closureBehaviour:       closureBehaviour,
	}, nil
}

func (c *PriorityConsumer[T]) NotifyClose(ch chan ClosedChannelEvent[T]) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()
	c.priorityChannel.NotifyClose(ch)
}

func (c *PriorityConsumer[T]) Consume() (<-chan Delivery[T], error) {
	fnGetResult := func(msg T, details ReceiveDetails) Delivery[T] {
		return Delivery[T]{Msg: msg, ReceiveDetails: details}
	}
	return doConsume(c, fnGetResult)
}

// ConsumeMessages returns a stream of just the message payloads (T only)
// while Consume returns a stream of Delivery[T] which includes the message payload and the receive details.
// This is useful when either you don't care about the receive details or they are already included in the message payload.
func (c *PriorityConsumer[T]) ConsumeMessages() (<-chan T, error) {
	fnGetResult := func(msg T, details ReceiveDetails) T {
		return msg
	}
	return doConsume(c, fnGetResult)
}

func doConsume[T any, R any](c *PriorityConsumer[T], fnGetResult func(msg T, details ReceiveDetails) R) (<-chan R, error) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.started {
		return nil, errors.New("consume already called")
	} else if c.isStopping {
		return nil, errors.New("cannot consume after stopping")
	}

	deliveries := make(chan R)
	c.started = true
	go func() {
		defer close(deliveries)
		for {
			// There is no context per-message, but there is a single context for the entire priority-channel
			// On receiving the message we do not pass any specific context,
			// but on processing the message we pass the priority-channel context
			msg, receiveDetails, status := c.priorityChannel.ReceiveWithContextEx(context.Background())
			channelName := receiveDetails.ChannelName

			if status != ReceiveSuccess {
				recoveryResult := tryAwaitRecovery(c.closureBehaviour, c, c.priorityChannel, status, channelName)
				if recoveryResult == awaitRecoveryNotApplicable {
					c.setClosed(status.ExitReason(), channelName)
					return
				}
				continue
			}

			select {
			case deliveries <- fnGetResult(msg, receiveDetails):
			case <-c.forceShutdownChannel:
				if c.onMessageDrop != nil {
					c.onMessageDrop(msg, channelName)
				}
				c.setClosed(PriorityChannelClosed, "")
				return
			}
		}
	}()

	return deliveries, nil
}

func (c *PriorityConsumer[T]) UpdatePriorityConfiguration(priorityConfiguration Configuration, innerPriorityChannelsContexts map[string]context.Context) error {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.isStopping {
		return errors.New("cannot update priority channel configuration after stopping")
	}

	err := c.priorityChannel.UpdatePriorityConfiguration(priorityConfiguration, innerPriorityChannelsContexts)
	if err != nil {
		return fmt.Errorf("failed to create priority channel from configuration: %w", err)
	}
	return nil
}

func (c *PriorityConsumer[T]) RecoverClosedInputChannel(channelName string, ch <-chan T) error {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.isStopping {
		return errors.New("cannot recover input channel after stopping")
	}

	c.priorityChannel.RecoverClosedInputChannel(channelName, ch)
	return nil
}

func (c *PriorityConsumer[T]) RecoverClosedPriorityChannel(channelName string, ctx context.Context) error {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	if c.isStopping {
		return errors.New("cannot recover priority channel after stopping")
	}

	c.priorityChannel.RecoverClosedPriorityChannel(channelName, ctx)
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
	}
	c.priorityChannelUpdatesMtx.Unlock()

	if !graceful {
		c.forceShutdownChannel <- struct{}{}
	}
	<-c.priorityChannelClosedC
}

type ProcessingStatus int

const (
	Running ProcessingStatus = iota
	Paused
	Stopped
)

// Status returns whether the consumer is stopped, and if so, the reason for stopping and,
// in case the reason is a closed channel, the name of the channel that was closed.
func (c *PriorityConsumer[T]) Status() (status ProcessingStatus, reason ExitReason, channelName string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	return c.status, c.exitReason, c.exitReasonChannelName
}

func (c *PriorityConsumer[T]) setPaused(exitReason ExitReason, exitReasonChannelName string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	c.status = Paused
	c.exitReason = exitReason
	c.exitReasonChannelName = exitReasonChannelName
}

func (c *PriorityConsumer[T]) setResumed() {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	c.status = Running
	c.exitReason = UnknownExitReason
	c.exitReasonChannelName = ""
}

func (c *PriorityConsumer[T]) setClosed(exitReason ExitReason, exitReasonChannelName string) {
	c.priorityChannelUpdatesMtx.Lock()
	defer c.priorityChannelUpdatesMtx.Unlock()

	c.status = Stopped
	c.exitReason = exitReason
	c.exitReasonChannelName = exitReasonChannelName
	close(c.priorityChannelClosedC)
}
