package channels

type ChannelWithWeight[T any, W any] struct {
	channelName string
	msgsC       <-chan T
	weight      W
}

func (c *ChannelWithWeight[T, W]) ChannelName() string {
	return c.channelName
}

func (c *ChannelWithWeight[T, W]) MsgsC() <-chan T {
	return c.msgsC
}

func (c *ChannelWithWeight[T, W]) Weight() W {
	return c.weight
}

func NewChannelWithWeight[T any, W any](channelName string, msgsC <-chan T, weight W) ChannelWithWeight[T, W] {
	return ChannelWithWeight[T, W]{
		channelName: channelName,
		msgsC:       msgsC,
		weight:      weight,
	}
}
