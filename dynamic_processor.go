package priority_channels

import (
	"context"
)

type DynamicPriorityProcessor[T any] struct {
	priorityChannel *PriorityChannel[T]
	workerPool      *DynamicWorkerPool[T]
}

func NewDynamicPriorityProcessor[T any](
	ctx context.Context,
	channelNameToChannel map[string]<-chan T,
	priorityConfiguration Configuration,
	workersNum int,
	closureBehaviour ClosureBehavior) (*DynamicPriorityProcessor[T], error) {
	priorityChannel, err := NewFromConfiguration(ctx, priorityConfiguration, channelNameToChannel)
	if err != nil {
		return nil, err
	}
	// We're closing the worker pool by having the consumer close its delivery channel.
	// We want the pool to continue processing messages until the consumer explicitly signals that it has stopped.
	// That's why we don't pass the same context used to cancel the consumer to the pool.
	workerPool, err := NewDynamicWorkerPool(context.Background(), priorityChannel, workersNum, closureBehaviour)
	if err != nil {
		return nil, err
	}

	processor := &DynamicPriorityProcessor[T]{
		priorityChannel: priorityChannel,
		workerPool:      workerPool,
	}
	return processor, nil
}

func (p *DynamicPriorityProcessor[T]) NotifyClose(ch chan ClosedChannelEvent[T]) {
	p.priorityChannel.NotifyClose(ch)
}

func (p *DynamicPriorityProcessor[T]) Process(processFn func(Delivery[T])) error {
	return p.workerPool.Process(processFn)
}

func (p *DynamicPriorityProcessor[T]) ProcessMessages(processFn func(T)) error {
	return p.workerPool.ProcessMessages(processFn)
}

func (p *DynamicPriorityProcessor[T]) Stop() {
	p.priorityChannel.Close()
	p.workerPool.Shutdown()
}

func (p *DynamicPriorityProcessor[T]) Done() <-chan struct{} {
	return p.workerPool.Done()
}

func (p *DynamicPriorityProcessor[T]) UpdatePriorityConfiguration(priorityConfiguration Configuration) error {
	return p.priorityChannel.UpdatePriorityConfiguration(priorityConfiguration)
}

func (p *DynamicPriorityProcessor[T]) RecoverClosedInputChannel(channelName string, ch <-chan T) {
	p.priorityChannel.RecoverClosedInputChannel(channelName, ch)
}

func (p *DynamicPriorityProcessor[T]) RecoverClosedPriorityChannel(channelName string, ctx context.Context) {
	p.priorityChannel.RecoverClosedPriorityChannel(channelName, ctx)
}

func (p *DynamicPriorityProcessor[T]) WorkersNum() int {
	return p.workerPool.WorkersNum()
}

func (p *DynamicPriorityProcessor[T]) UpdateWorkersNum(newWorkersNum int) error {
	return p.workerPool.UpdateWorkersNum(newWorkersNum)
}

func (p *DynamicPriorityProcessor[T]) ActiveWorkersNum() int {
	return p.workerPool.ActiveWorkersNum()
}

func (p *DynamicPriorityProcessor[T]) Status() (stopped bool, reason ExitReason, channelName string) {
	return p.workerPool.Status()
}
