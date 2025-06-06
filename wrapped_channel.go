package priority_channels

import (
	"context"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func WrapAsPriorityChannel[T any](ctx context.Context, channelName string, msgsC <-chan T, options ...func(*PriorityChannelOptions)) (*PriorityChannel[T], error) {
	if channelName == "" {
		return nil, ErrEmptyChannelName
	}
	pcOptions := &PriorityChannelOptions{}
	for _, option := range options {
		option(pcOptions)
	}
	compositeChannel := &wrappedChannel[T]{
		channelName:                channelName,
		msgsC:                      msgsC,
		autoDisableOnClosedChannel: pcOptions.autoDisableClosedChannels,
	}
	channelNameToChannel := map[string]<-chan T{
		channelName: msgsC,
	}
	return newPriorityChannel(ctx, compositeChannel, channelNameToChannel, options...), nil
}

type wrappedChannel[T any] struct {
	channelName                string
	msgsC                      <-chan T
	autoDisableOnClosedChannel bool
	disabled                   bool
}

func (w *wrappedChannel[T]) ChannelName() string {
	return w.channelName
}

func (w *wrappedChannel[T]) NextSelectCases(upto int) (selectCases []selectable.SelectCase[T], isLastIteration bool, closedChannel *selectable.ClosedChannelDetails) {
	if w.disabled {
		return nil, true, nil
	}
	return []selectable.SelectCase[T]{
		{
			ChannelName: w.channelName,
			MsgsC:       w.msgsC,
		},
	}, true, nil
}

func (c *wrappedChannel[T]) UpdateOnCaseSelected(pathInTree []selectable.ChannelNode, recvOK bool) {
	if !recvOK && c.autoDisableOnClosedChannel {
		c.disabled = true
	}
}

func (c *wrappedChannel[T]) RecoverClosedInputChannel(ch <-chan T, pathInTree []selectable.ChannelNode) {
	if c.disabled {
		c.disabled = false
	}
	c.msgsC = ch
}

func (c *wrappedChannel[T]) RecoverClosedInnerPriorityChannel(ctx context.Context, pathInTree []selectable.ChannelNode) {
}

func (c *wrappedChannel[T]) GetInputAndInnerPriorityChannels(inputChannels map[string]<-chan T, innerPriorityChannels map[string]context.Context) error {
	if _, ok := inputChannels[c.channelName]; ok {
		return &DuplicateChannelError{ChannelName: c.channelName}
	}
	inputChannels[c.channelName] = c.msgsC
	return nil
}

func (c *wrappedChannel[T]) GetInputChannelsPaths(m map[string][]selectable.ChannelNode, currPathInTree []selectable.ChannelNode) {
	if _, ok := m[c.channelName]; !ok {
		return
	}
	m[c.channelName] = currPathInTree
}

func (c *wrappedChannel[T]) Clone() selectable.Channel[T] {
	return &wrappedChannel[T]{
		channelName:                c.channelName,
		msgsC:                      c.msgsC,
		autoDisableOnClosedChannel: c.autoDisableOnClosedChannel,
		disabled:                   c.disabled,
	}
}

func (c *wrappedChannel[T]) Validate() error {
	if c.channelName == "" {
		return ErrEmptyChannelName
	}
	return nil
}
