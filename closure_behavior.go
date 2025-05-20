package priority_channels

import "context"

type ClosureBehavior struct {
	InputChannelClosureBehavior    ChannelClosureBehavior
	PriorityChannelClosureBehavior ChannelClosureBehavior
	NoOpenChannelsBehavior         NoOpenChannelsBehavior
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
	awaitRecoveryCancelled
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
		return awaitRecoveryCancelled
	case status == ReceivePriorityChannelClosed && channelName != "" && behaviour.PriorityChannelClosureBehavior == PauseOnClosed:
		pauser.setPaused(status.ExitReason(), channelName)
		if priorityChannel.AwaitRecover(context.Background(), channelName, PriorityChannelType) {
			pauser.setResumed()
			return awaitRecoverySuccess
		}
		return awaitRecoveryCancelled
	case status == ReceiveNoOpenChannels && behaviour.NoOpenChannelsBehavior == PauseWhenNoOpenChannels:
		pauser.setPaused(status.ExitReason(), "")
		if priorityChannel.AwaitOpenChannel(context.Background()) {
			pauser.setResumed()
			return awaitRecoverySuccess
		}
		return awaitRecoveryCancelled
	default:
		return awaitRecoveryNotApplicable
	}
}
