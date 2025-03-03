package selectable

type Channel[T any] interface {
	ChannelName() string
	NextSelectCases(upto int) (selectCases []SelectCase[T], isLastIteration bool, closedChannel *ClosedChannelDetails)
	UpdateOnCaseSelected(pathInTree []ChannelNode, recvOK bool)
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
