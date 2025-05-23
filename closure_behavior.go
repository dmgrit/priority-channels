package priority_channels

import "context"

type ClosureBehavior struct {
	InputChannelClosureBehavior         ChannelClosureBehavior
	InnerPriorityChannelClosureBehavior ChannelClosureBehavior
	NoOpenChannelsBehavior              NoOpenChannelsBehavior
}

type ChannelClosureBehavior int

const (
	StopOnClosed ChannelClosureBehavior = iota
	PauseOnClosed
)

type NoOpenChannelsBehavior int

const (
	StopWhenNoOpenChannels NoOpenChannelsBehavior = iota
	PauseWhenNoOpenChannels
)

type pauseAndResumer interface {
	setPaused(reason ExitReason, channelName string)
	setResumed()
}

type awaitRecoveryResult int

const (
	awaitRecoverySuccess awaitRecoveryResult = iota
	awaitRecoveryCanceled
	awaitRecoveryNotApplicable
)

func tryAwaitRecovery[T any](behaviour ClosureBehavior, pauser pauseAndResumer, priorityChannel *PriorityChannel[T], status ReceiveStatus, channelName string) awaitRecoveryResult {
	switch {
	case status == ReceiveChannelClosed && behaviour.InputChannelClosureBehavior == PauseOnClosed:
		pauser.setPaused(status.ExitReason(), channelName)
		if priorityChannel.AwaitRecover(context.Background(), channelName, InputChannelType) {
			pauser.setResumed()
			return awaitRecoverySuccess
		}
		return awaitRecoveryCanceled
	case status == ReceivePriorityChannelClosed && channelName != "" && behaviour.InnerPriorityChannelClosureBehavior == PauseOnClosed:
		pauser.setPaused(status.ExitReason(), channelName)
		if priorityChannel.AwaitRecover(context.Background(), channelName, InnerPriorityChannelType) {
			pauser.setResumed()
			return awaitRecoverySuccess
		}
		return awaitRecoveryCanceled
	case status == ReceiveNoOpenChannels && behaviour.NoOpenChannelsBehavior == PauseWhenNoOpenChannels:
		pauser.setPaused(status.ExitReason(), "")
		if priorityChannel.AwaitOpenChannel(context.Background()) {
			pauser.setResumed()
			return awaitRecoverySuccess
		}
		return awaitRecoveryCanceled
	default:
		return awaitRecoveryNotApplicable
	}
}
