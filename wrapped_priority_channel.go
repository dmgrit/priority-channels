package priority_channels

import (
	"context"
	"fmt"

	"github.com/dmgrit/priority-channels/internal/selectable"
)

func asSelectableChannelWithName[T any](pc *PriorityChannel[T], name string) selectable.Channel[T] {
	return &overrideCompositeChannelName[T]{
		ctx:     pc.ctx,
		name:    name,
		channel: pc.compositeChannel,
	}
}

type overrideCompositeChannelName[T any] struct {
	ctx     context.Context
	name    string
	channel selectable.Channel[T]
}

func (oc *overrideCompositeChannelName[T]) ChannelName() string {
	return oc.name
}

func (oc *overrideCompositeChannelName[T]) NextSelectCases(upto int) ([]selectable.SelectCase[T], bool, *selectable.ClosedChannelDetails) {
	select {
	case <-oc.ctx.Done():
		return nil, true, &selectable.ClosedChannelDetails{
			ChannelName: oc.ChannelName(),
			PathInTree:  nil,
		}
	default:
		res, allSelected, closedChannel := oc.channel.NextSelectCases(upto)
		if closedChannel != nil {
			if len(closedChannel.PathInTree) > 0 {
				closedChannel.PathInTree[len(closedChannel.PathInTree)-1].ChannelName = oc.name
			} else {
				closedChannel.ChannelName = oc.name
			}
			return res, allSelected, closedChannel
		}
		for i, sc := range res {
			if len(sc.PathInTree) > 0 {
				res[i].PathInTree[len(res[i].PathInTree)-1].ChannelName = oc.name
			}
		}
		return res, allSelected, nil
	}
}

func (oc *overrideCompositeChannelName[T]) UpdateOnCaseSelected(pathInTree []selectable.ChannelNode, recvOK bool) {
	oc.channel.UpdateOnCaseSelected(pathInTree, recvOK)
}

func (oc *overrideCompositeChannelName[T]) RecoverClosedInputChannel(ch <-chan T, pathInTree []selectable.ChannelNode) {
	oc.channel.RecoverClosedInputChannel(ch, pathInTree)
}

func (oc *overrideCompositeChannelName[T]) RecoverClosedInnerPriorityChannel(ctx context.Context, pathInTree []selectable.ChannelNode) {
	if len(pathInTree) == 0 {
		oc.ctx = ctx
		return
	}
	oc.channel.RecoverClosedInnerPriorityChannel(ctx, pathInTree)
}

func (oc *overrideCompositeChannelName[T]) GetInputAndInnerPriorityChannels(inputChannels map[string]<-chan T, innerPriorityChannels map[string]context.Context) error {
	if _, ok := innerPriorityChannels[oc.name]; ok {
		return fmt.Errorf("priority channel name '%s' is used more than once", oc.name)
	}
	innerPriorityChannels[oc.name] = oc.ctx
	return oc.channel.GetInputAndInnerPriorityChannels(inputChannels, innerPriorityChannels)
}

func (oc *overrideCompositeChannelName[T]) GetInputChannelsPaths(m map[string][]selectable.ChannelNode, currPathInTree []selectable.ChannelNode) {
	oc.channel.GetInputChannelsPaths(m, currPathInTree)
}

func (oc *overrideCompositeChannelName[T]) Clone() selectable.Channel[T] {
	return &overrideCompositeChannelName[T]{
		ctx:     oc.ctx,
		name:    oc.name,
		channel: oc.channel.Clone(),
	}
}
