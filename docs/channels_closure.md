# Channels Closure Handling

This library defines three types of channels:   
1. **Input Go Channels** - These are channels that receive messages from external sources. The library does not control their closure, but it can handle closures gracefully. Input channels are the **leaves** of the channel tree.
2. **Inner Priority Channels** -  These are internal channels that form the intermediate **nodes** of the channel tree. They apply prioritization rules and may have their own cancellable context.
3. **Root Priority Channel** - This is the **top-level** priority channel and serves as the main interface for receiving messages.

## Channel Closure Scenarios

### Root Priority Channel Closure
When the root priority channel is closed, it signifies that the entire channel tree is closed. No further messages can be received.
Any subsequent call to `Receive()` will return a `ReceivePriorityChannelClosed` status.

### Handling Other Closure Scenarios
* Closure of input Go channels
* Closure of inner priority channels (via their context being canceled or an explicit `Close()` call)

In these cases, further `Receive()` calls will return either:
* `ReceiveInputChannelClosed` or 
* `ReceiveInnerPriorityChannelClosed`   
unless the parent priority channel has `AutoDisableClosedChannels` set to true.
In that case, the closed channel is automatically disabled, and messages continue flowing from other active channels in the tree. No closure status is returned by `Receive()` in this scenario..

### When No Receivable Path Remains
If closures result in **no remaining path** from the root to a live input channel:
- In flat mode, this happens when all input channels are closed
- In hierarchical mode, this can happen due to a combination of closed input and inner priority channels.
In both cases, `Receive()` returns a `ReceiveNoReceivablePath` status.

## Notification and Recovery

### Notifications on Channel Closure
You can register to receive closure notifications regardless of the `AutoDisableClosedChannels` setting, using:
```go
NotifyClose(ch chan ClosedChannelEvent[T])
```
The `ClosedChannelEvent[T]` struct includes:
* The name of the closed channel
* Its type (`InputChannel` or `InnerPriorityChannel`)
* The path to the channel (`ReceiveDetails`)

### Channel Recovery
You can recover closed channels as follows:
```go
RecoverClosedInputChannel(name string, ch <-chan T)
RecoverClosedInnerPriorityChannel(name string, ctx context.Context)
```
* `RecoverClosedInputChannel()` attaches a new **Go channel** as a replacement.
* `RecoverClosedInnerPriorityChannel()` reattaches a **new context** to a previously closed inner priority channel.


### Awaiting Recovery or Receivable Path
You can wait for channel recovery or the restoration of a receivable path:
```go
AwaitRecover(ctx context.Context, name string, channelType ChannelType) bool
AwaitReceivablePath(ctx context.Context) bool
```
* `AwaitRecover` blocks until the specified channel is recovered or the context is canceled..
* `AwaitReceivablePath` blocks until a receivable path is restored or the context is canceled.


### Configuration for Consumers and Worker Pools
`Consumer` and `DynamicWorkerPool`, which are built on top of the priority channel system, require explicit configuration of closure behavior at initialization:
```go
type ClosureBehavior struct {
    InputChannelClosureBehavior         ChannelClosureBehavior
    InnerPriorityChannelClosureBehavior ChannelClosureBehavior
    NoReceivablePathBehavior            NoReceivablePathBehavior
}

type ChannelClosureBehavior int

const (
    StopOnClosed ChannelClosureBehavior = iota
    PauseOnClosed
)

type NoReceivablePathBehavior int

const (
    StopWhenNoReceivablePath NoReceivablePathBehavior = iota
    PauseWhenNoReceivablePath
)
```
`Stop` means that the component will cease processing messages permanently when a closure or lack of receivable path is detected.  
`Pause` means that processing will temporarily stop, waiting for recovery or restoration of the path before resuming.

 

