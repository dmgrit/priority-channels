package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var inputChannels []chan string
	var triggerPauseChannels []chan bool
	var triggerCloseChannels []chan bool
	priorityChannelsCancelFuncs := make(map[string]context.CancelFunc)

	channelsNum := 5
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseChannels = append(triggerPauseChannels, make(chan bool))
		triggerCloseChannels = append(triggerCloseChannels, make(chan bool))
	}

	var options []func(*priority_channels.PriorityChannelOptions)
	options = append(options, priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully))
	if len(os.Args) > 1 && os.Args[1] == "-a" {
		options = append(options, priority_channels.AutoDisableClosedChannels())
	}
	customerAPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Customer A - High Priority",
			inputChannels[0],
			3),
		channels.NewChannelWithFreqRatio(
			"Customer A - Low Priority",
			inputChannels[1],
			1),
	}, options...)
	if err != nil {
		fmt.Printf("Unexpected error on priority channel intialization: %v\n", err)
		return
	}

	customerBPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Customer B - High Priority",
			inputChannels[2],
			3),
		channels.NewChannelWithFreqRatio(
			"Customer B - Low Priority",
			inputChannels[3],
			1),
	}, options...)
	if err != nil {
		fmt.Printf("Unexpected error on priority channel intialization: %v\n", err)
		return
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio("Customer A",
			customerAPriorityChannel,
			5),
		priority_channels.NewPriorityChannelWithFreqRatio("Customer B",
			customerBPriorityChannel,
			1),
	}

	combinedUsersAndMessageTypesPriorityChannel, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio, options...)
	if err != nil {
		fmt.Printf("Unexpected error on priority channel intialization: %v\n", err)
		return
	}

	urgentMessagesPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, "Urgent Messages", inputChannels[4], options...)
	if err != nil {
		fmt.Printf("failed to create urgent message priority channel: %v\n", err)
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst(ctx, []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority(
			"Customer Messages",
			combinedUsersAndMessageTypesPriorityChannel,
			1),
		priority_channels.NewPriorityChannelWithPriority(
			"Urgent Messages",
			urgentMessagesPriorityChannel,
			100),
	}, options...)
	if err != nil {
		fmt.Printf("Unexpected error on priority channel intialization: %v\n", err)
		return
	}

	demoFilePath := filepath.Join(os.TempDir(), "priority_channels_demo.txt")

	fmt.Printf("Multi-Hierarchy Demo:\n")
	fmt.Printf("- Press 'A/NA' to start/stop receiving messages from Customer A\n")
	fmt.Printf("- Press 'B/NB' to start/stop receiving messages from Customer B\n")
	fmt.Printf("- Press 'H/NH' to start/stop receiving high priority messages\n")
	fmt.Printf("- Press 'L/NL' to start/stop receiving low priority messages\n")
	fmt.Printf("- Press 'U/NU' to start/stop receiving urgent messages\n")
	fmt.Printf("- Press 'D/ND' to start/stop presenting receive path in tree\n")
	fmt.Printf("- Press 0 to exit\n\n")
	fmt.Printf("To see the results live, run in another terminal window:\ntail -f %s\n\n", demoFilePath)

	for i := 1; i <= len(inputChannels); i++ {
		go func(i int) {
			paused := true
			closed := false
			for {
				select {
				case b := <-triggerPauseChannels[i-1]:
					paused = !b
				case b := <-triggerCloseChannels[i-1]:
					if b && !closed {
						close(inputChannels[i-1])
						closed = true
					}
				default:
					if !paused && !closed {
						select {
						case b := <-triggerPauseChannels[i-1]:
							paused = !b
						case b := <-triggerCloseChannels[i-1]:
							if b && !closed {
								close(inputChannels[i-1])
								closed = true
							}
						case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
						}
					} else {
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}(i)
	}

	var presentDetails atomic.Bool

	go func() {
		f, err := os.Create(demoFilePath)
		if err != nil {
			fmt.Printf("Failed to open file: %v\n", err)
			cancel()
			return
		}
		defer f.Close()
		prevFullChannelPath := ""
		streakLength := 0

		for {
			ctx := context.Background()
			_, details, status := ch.ReceiveWithContextEx(ctx)
			fullChannelPath := ""
			if presentDetails.Load() {
				for _, channelNode := range details.PathInTree {
					fullChannelPath += fmt.Sprintf("%s [%d] -> ", channelNode.ChannelName, channelNode.ChannelIndex)
				}
				fullChannelPath = fullChannelPath + fmt.Sprintf("%s [%d]", details.ChannelName, details.ChannelIndex)
			} else {
				fullChannelPath = details.ChannelName
			}
			if status == priority_channels.ReceiveSuccess {
				if fullChannelPath == prevFullChannelPath {
					streakLength++
				} else {
					streakLength = 1
				}
				prevFullChannelPath = fullChannelPath
				logMessage := fmt.Sprintf("%s (%d)\n", fullChannelPath, streakLength)

				_, err := f.WriteString(logMessage)
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceiveInputChannelClosed {
				_, err := f.WriteString(fmt.Sprintf("Channel '%s' is closed\n", fullChannelPath))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceiveInnerPriorityChannelClosed {
				_, err = f.WriteString(fmt.Sprintf("Inner Priority Channel '%s' is closed\n", fullChannelPath))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceivePriorityChannelClosed {
				_, err = f.WriteString(fmt.Sprintf("Priority Channel is closed\n"))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceiveNoReceivablePath {
				_, err := f.WriteString("No receivable path left\n")
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else {
				_, err := f.WriteString(fmt.Sprintf("Unexpected status %s\n", fullChannelPath))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			}

			if status != priority_channels.ReceiveSuccess &&
				status != priority_channels.ReceiveInputChannelClosed &&
				status != priority_channels.ReceiveInnerPriorityChannelClosed {
				_, err := f.WriteString("Exiting\n")
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		upperLine := strings.ToUpper(line)
		value := !strings.HasPrefix(upperLine, "N")
		operation := "Started"
		if !value {
			operation = "Stopped"
		}
		if strings.HasPrefix(upperLine, "C") {
			switch upperLine {
			case "CA":
				fmt.Printf("Closing Priority Channel of Customer A\n")
				if cancelFunc, ok := priorityChannelsCancelFuncs["Customer A"]; ok {
					cancelFunc()
					delete(priorityChannelsCancelFuncs, "Customer A")
				} else {
					customerAPriorityChannel.Close()
				}
				continue
			case "CB":
				fmt.Printf("Closing Priority Channel of Customer B\n")
				if cancelFunc, ok := priorityChannelsCancelFuncs["Customer B"]; ok {
					cancelFunc()
					delete(priorityChannelsCancelFuncs, "Customer B")
				} else {
					customerBPriorityChannel.Close()
				}
				continue
			case "CU":
				fmt.Printf("Closing Priority Channel of Urgent Messages\n")
				if cancelFunc, ok := priorityChannelsCancelFuncs["Urgent Messages"]; ok {
					cancelFunc()
					delete(priorityChannelsCancelFuncs, "Urgent Messages")
				} else {
					urgentMessagesPriorityChannel.Close()
				}
				continue
			case "CC":
				fmt.Printf("Closing Combined Priority Channel of Both Customers\n")
				if cancelFunc, ok := priorityChannelsCancelFuncs["Customer Messages"]; ok {
					cancelFunc()
					delete(priorityChannelsCancelFuncs, "Customer Messages")
				} else {
					combinedUsersAndMessageTypesPriorityChannel.Close()
				}
				continue
			case "CG":
				fmt.Printf("Closing Priority Channel \n")
				ch.Close()
				continue
			}
			upperLine = strings.TrimPrefix(upperLine, "C")
			number, err := strconv.Atoi(upperLine)
			if err != nil || number <= 0 || number > channelsNum {
				continue
			}
			fmt.Printf("Closing Channel %d\n", number)
			triggerCloseChannels[number-1] <- value
			continue
		}

		switch upperLine {
		case "A", "NA":
			triggerPauseChannels[0] <- value
			triggerPauseChannels[1] <- value
			fmt.Printf(operation + " receiving messages for Customer A\n")
		case "B", "NB":
			triggerPauseChannels[2] <- value
			triggerPauseChannels[3] <- value
			fmt.Printf(operation + " receiving messages for Customer B\n")
		case "H", "NH":
			triggerPauseChannels[0] <- value
			triggerPauseChannels[2] <- value
			fmt.Printf(operation + " receiving High Priority messages\n")
		case "L", "NL":
			triggerPauseChannels[1] <- value
			triggerPauseChannels[3] <- value
			fmt.Printf(operation + " receiving Low Priority messages\n")
		case "U", "NU":
			triggerPauseChannels[4] <- value
			fmt.Printf(operation + " receiving Urgent messages\n")
		case "RA":
			fmt.Printf("Recovering Priority Channel of Customer A\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			priorityChannelsCancelFuncs["Customer A"] = cancelFunc
			ch.RecoverClosedInnerPriorityChannel("Customer A", newCtx)
		case "RB":
			fmt.Printf("Recovering Priority Channel of Customer B\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			priorityChannelsCancelFuncs["Customer B"] = cancelFunc
			ch.RecoverClosedInnerPriorityChannel("Customer B", newCtx)
		case "RU":
			fmt.Printf("Recovering Priority Channel of Urgent Messages\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			priorityChannelsCancelFuncs["Urgent Messages"] = cancelFunc
			ch.RecoverClosedInnerPriorityChannel("Urgent Messages", newCtx)
		case "RCC":
			fmt.Printf("Recovering Combined Priority Channel of Both Customers\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			priorityChannelsCancelFuncs["Customer Messages"] = cancelFunc
			ch.RecoverClosedInnerPriorityChannel("Customer Messages", newCtx)
		case "D":
			presentDetails.Store(true)
		case "ND":
			presentDetails.Store(false)
		case "0":
			fmt.Printf("Exiting\n")
			cancel()
			return
		}
	}
}
