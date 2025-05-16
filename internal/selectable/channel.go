package selectable

import "context"

type Channel[T any] interface {
	ChannelName() string
	NextSelectCases(upto int) (selectCases []SelectCase[T], isLastIteration bool, closedChannel *ClosedChannelDetails)
	UpdateOnCaseSelected(pathInTree []ChannelNode, recvOK bool)
	RecoverClosedInputChannel(ch <-chan T, pathInTree []ChannelNode)
	RecoverClosedPriorityChannel(ctx context.Context, pathInTree []ChannelNode)
	GetInputChannels(map[string]<-chan T) error
	GetInputChannelsPaths(result map[string][]ChannelNode, currPathInTree []ChannelNode)
	Clone() Channel[T]
}

type SelectCase[T any] struct {
	MsgsC       <-chan T
	ChannelName string
	PathInTree  []ChannelNode
}

type ClosedChannelDetails struct {
	ChannelName string
	PathInTree  []ChannelNode
}

type ChannelNode struct {
	ChannelName  string
	ChannelIndex int
}
