package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmgrit/priority-channels"
)

type channelStatsMessage struct {
	ChannelName string
	Message     string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var inputChannels []chan string
	var triggerPauseChannels []chan bool
	var triggerCloseChannels []chan bool
	var triggerRecoverChannels []chan chan string

	channelsNum := 5
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseChannels = append(triggerPauseChannels, make(chan bool))
		triggerCloseChannels = append(triggerCloseChannels, make(chan bool))
		triggerRecoverChannels = append(triggerRecoverChannels, make(chan chan string))
	}
	channelsOrder := map[string]int{
		"Customer A - High Priority": 1,
		"Customer A - Low Priority":  2,
		"Customer B - High Priority": 3,
		"Customer B - Low Priority":  4,
		"Urgent Messages":            5,
	}

	priorityConfig := priority_channels.Configuration{
		PriorityChannel: &priority_channels.PriorityChannelConfig{
			Method: priority_channels.ByHighestAlwaysFirstMethodConfig,
			Channels: []priority_channels.ChannelConfig{
				{
					Name:     "Urgent Messages",
					Priority: 10,
				},
				{
					Name:     "Customer Messages",
					Priority: 1,
					PriorityChannelConfig: &priority_channels.PriorityChannelConfig{
						Method: priority_channels.ByFrequencyRatioMethodConfig,
						Channels: []priority_channels.ChannelConfig{
							{
								Name:      "Customer A",
								FreqRatio: 5,
								PriorityChannelConfig: &priority_channels.PriorityChannelConfig{
									Method: priority_channels.ByFrequencyRatioMethodConfig,
									Channels: []priority_channels.ChannelConfig{
										{
											Name:      "Customer A - High Priority",
											FreqRatio: 3,
										},
										{
											Name:      "Customer A - Low Priority",
											FreqRatio: 1,
										},
									},
								},
							},
							{
								Name:      "Customer B",
								FreqRatio: 1,
								PriorityChannelConfig: &priority_channels.PriorityChannelConfig{
									Method: priority_channels.ByFrequencyRatioMethodConfig,
									Channels: []priority_channels.ChannelConfig{
										{
											Name:      "Customer B - High Priority",
											FreqRatio: 3,
										},
										{
											Name:      "Customer B - Low Priority",
											FreqRatio: 1,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if len(os.Args) > 1 && os.Args[1] == "-a" {
		setAutoDisabledPriorityChannelConfig(priorityConfig.PriorityChannel)
	}

	channelNameToChannel := map[string]<-chan string{
		"Customer A - High Priority": inputChannels[0],
		"Customer A - Low Priority":  inputChannels[1],
		"Customer B - High Priority": inputChannels[2],
		"Customer B - Low Priority":  inputChannels[3],
		"Urgent Messages":            inputChannels[4],
	}
	channelNames := []string{
		"Customer A - High Priority",
		"Customer A - Low Priority",
		"Customer B - High Priority",
		"Customer B - Low Priority",
		"Urgent Messages",
	}

	receivedMsgs := 0
	byChannelName := make(map[string]int)
	var receivedMsgsMutex sync.Mutex
	var presentDetails atomic.Bool

	processFn := func(d priority_channels.Delivery[string]) {
		time.Sleep(100 * time.Millisecond)
		receivedMsgsMutex.Lock()
		receivedMsgs++
		fullChannelPath := ""
		if presentDetails.Load() {
			for _, channelNode := range d.ReceiveDetails.PathInTree {
				fullChannelPath += fmt.Sprintf("%s [%d] -> ", channelNode.ChannelName, channelNode.ChannelIndex)
			}
			fullChannelPath = fullChannelPath + fmt.Sprintf("%s [%d]", d.ReceiveDetails.ChannelName, d.ReceiveDetails.ChannelIndex)
		} else {
			fullChannelPath = d.ReceiveDetails.ChannelName
		}
		byChannelName[fullChannelPath] = byChannelName[fullChannelPath] + 1
		receivedMsgsMutex.Unlock()
	}
	closureBehavior := priority_channels.ClosureBehavior{
		InputChannelClosureBehavior:         priority_channels.PauseOnClosed,
		InnerPriorityChannelClosureBehavior: priority_channels.PauseOnClosed,
		NoOpenChannelsBehavior:              priority_channels.PauseWhenNoOpenChannels,
	}

	innerPriorityChannelsContexts, innerPriorityChannelsCancelFuncs := generateInnerPriorityChannelsContextsAndCancelFuncs()
	wp, err := priority_channels.NewDynamicPriorityProcessor(ctx, channelNameToChannel, innerPriorityChannelsContexts, priorityConfig, 3, closureBehavior)
	if err != nil {
		fmt.Printf("failed to initialize dynamic priority processor: %v\n", err)
		os.Exit(1)
	}

	err = wp.Process(processFn)
	if err != nil {
		fmt.Printf("failed to start processing messages: %v\n", err)
		os.Exit(1)
	}

	demoFilePath := filepath.Join(os.TempDir(), "priority_channels_demo.txt")
	f, err := os.Create(demoFilePath)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		cancel()
		return
	}
	defer f.Close()

	fmt.Printf("Dynamic-Priority-Processor Multi-Hierarchy Demo:\n")
	fmt.Printf("- Press 'A/NA' to start/stop receiving messages from Customer A\n")
	fmt.Printf("- Press 'B/NB' to start/stop receiving messages from Customer B\n")
	fmt.Printf("- Press 'H/NH' to start/stop receiving high priority messages\n")
	fmt.Printf("- Press 'L/NL' to start/stop receiving low priority messages\n")
	fmt.Printf("- Press 'U/NU' to start/stop receiving urgent messages\n")
	fmt.Printf("- Press 'D/ND' to start/stop presenting receive path in tree\n")
	fmt.Printf("- Press 'w <workers_num>' to update number of workers in the worker pool\n")
	fmt.Printf("- Press 'load <priority_configuration_file>' to load a different priority configuration\n")
	fmt.Printf("- Press 'w' or 'workers' to see the number of worker goroutines configured in the worker pool\n")
	fmt.Printf("- Press 'active' to see the number of goroutines currently processing messages\n")
	fmt.Printf("- Press quit to exit\n\n")
	fmt.Printf("To see the results live, run in another terminal window:\ntail -f %s\n\n", demoFilePath)

	for i := 1; i <= len(inputChannels); i++ {
		go func(inputChannel chan string, triggerPauseChannel chan bool, triggerCloseChannel chan bool, triggerRecoverChannel chan chan string) {
			paused := true
			closed := false
			j := 0
			for {
				j++
				select {
				case b := <-triggerPauseChannel:
					paused = !b
				case b := <-triggerCloseChannel:
					if b && !closed {
						close(inputChannel)
						closed = true
					}
				case inputChannel = <-triggerRecoverChannel:
					closed = false
				default:
					if !paused && !closed {
						select {
						case b := <-triggerPauseChannel:
							paused = !b
						case b := <-triggerCloseChannel:
							if b && !closed {
								close(inputChannel)
								closed = true
							}
						case inputChannel <- fmt.Sprintf("message-%d", j):
						}
					} else {
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}(inputChannels[i-1], triggerPauseChannels[i-1], triggerCloseChannels[i-1], triggerRecoverChannels[i-1])
	}

	tickerCh := time.Tick(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tickerCh:
				receivedMsgsMutex.Lock()
				separatorLen := 80
				if presentDetails.Load() {
					separatorLen = 100
				}
				_, _ = f.WriteString(strings.Repeat("=", separatorLen) + "\n")
				if receivedMsgs > 0 {
					_, _ = f.WriteString(fmt.Sprintf("Received %d messages in the last 5 seconds (%.2f messages per second)\n", receivedMsgs, float64(receivedMsgs)/5))
					var messages []channelStatsMessage
					for name, val := range byChannelName {
						messages = append(messages, channelStatsMessage{
							ChannelName: name,
							Message:     fmt.Sprintf("%d messages (%.2f%%)", val, float64(val)/float64(receivedMsgs)*100),
						})
					}
					sort.Slice(messages, func(i, j int) bool {
						return channelsOrder[messages[i].ChannelName] < channelsOrder[messages[j].ChannelName]
					})
					for _, msg := range messages {
						_, _ = f.WriteString(fmt.Sprintf("%s: %s\n", msg.ChannelName, msg.Message))
					}
				} else {
					_, _ = f.WriteString("No messages received in the last 5 seconds\n")
					status, reason, channelName := wp.Status()
					if status != priority_channels.Running {
						var state string
						if status == priority_channels.Stopped {
							state = "stopped"
						} else {
							state = "paused"
						}
						switch reason {
						case priority_channels.UnknownExitReason:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool %s: Unknown reason\n", state))
						case priority_channels.ChannelClosed:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool %s: Channel '%s' closed\n", state, channelName))
						case priority_channels.PriorityChannelClosed:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool %s: Priority Channel '%s' closed\n", state, channelName))
						case priority_channels.NoOpenChannels:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool %s: No open channels\n", state))
						case priority_channels.ContextCanceled:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool %s: Context canceled\n", state))
						}
					}
				}
				_, _ = f.WriteString(strings.Repeat("=", separatorLen) + "\n")

				receivedMsgs = 0
				byChannelName = make(map[string]int)
				receivedMsgsMutex.Unlock()
			}
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
				if cancelFunc := innerPriorityChannelsCancelFuncs["Customer A"]; cancelFunc != nil {
					fmt.Printf("Closing Priority Channel of Customer A\n")
					cancelFunc()
				}
				continue
			case "CB":
				if cancelFunc := innerPriorityChannelsCancelFuncs["Customer B"]; cancelFunc != nil {
					fmt.Printf("Closing Priority Channel of Customer B\n")
					cancelFunc()
				}
				continue
			case "CU":
				if cancelFunc := innerPriorityChannelsCancelFuncs["Urgent Messages"]; cancelFunc != nil {
					fmt.Printf("Closing Priority Channel of Urgent Messages\n")
					cancelFunc()
				}
				continue
			case "CC":
				if cancelFunc := innerPriorityChannelsCancelFuncs["Customer Messages"]; cancelFunc != nil {
					fmt.Printf("Closing Priority Channel of of Both Customers\n")
					cancelFunc()
				}
				continue
			}
			upperLine = strings.TrimPrefix(upperLine, "C")
			number, err := strconv.Atoi(upperLine)
			if err != nil || number <= 0 || number > channelsNum {
				continue
			}
			fmt.Printf("Closing Channel '%s'\n", channelNames[number-1])
			triggerCloseChannels[number-1] <- value
			continue
		}

		words := strings.Split(line, " ")
		if len(words) == 2 {
			switch words[0] {
			case "load":
				contents, err := os.ReadFile(words[1])
				if err != nil {
					fmt.Printf("failed to load file %s: %v\n", words[1], err)
					continue
				}

				var priorityConfig priority_channels.Configuration
				if err := json.Unmarshal(contents, &priorityConfig); err != nil {
					fmt.Printf("failed to unmarshal priority configuration: %v\n", err)
					continue
				}

				innerPriorityChannelsContexts, innerPriorityChannelsCancelFuncs = generateInnerPriorityChannelsContextsAndCancelFuncs()
				if err := wp.UpdatePriorityConfiguration(priorityConfig, innerPriorityChannelsContexts); err != nil {
					fmt.Printf("failed to update priority consumer configuration: %v\n", err)
					continue
				}
				fmt.Printf("Updated prioritization configuration\n")
			case "workers", "w":
				num, err := strconv.Atoi(words[1])
				if err != nil || num < 0 {
					fmt.Printf("second parameter to workers should be a non-negative number\n")
					continue
				}
				if err := wp.UpdateWorkersNum(num); err != nil {
					fmt.Printf("failed to update workers number: %v\n", err)
					continue
				}
				fmt.Printf("Updated workers number to %d\n", num)
			default:
				continue
			}
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
			innerPriorityChannelsCancelFuncs["Customer A"] = cancelFunc
			wp.RecoverClosedInnerPriorityChannel("Customer A", newCtx)
		case "RB":
			fmt.Printf("Recovering Priority Channel of Customer B\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			innerPriorityChannelsCancelFuncs["Customer B"] = cancelFunc
			wp.RecoverClosedInnerPriorityChannel("Customer B", newCtx)
		case "RU":
			fmt.Printf("Recovering Priority Channel of Urgent Messages\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			innerPriorityChannelsCancelFuncs["Urgent Messages"] = cancelFunc
			wp.RecoverClosedInnerPriorityChannel("Urgent Messages", newCtx)
		case "RCC":
			fmt.Printf("Recovering Combined Priority Channel of Both Customers\n")
			newCtx, cancelFunc := context.WithCancel(context.Background())
			innerPriorityChannelsCancelFuncs["Customer Messages"] = cancelFunc
			wp.RecoverClosedInnerPriorityChannel("Customer Messages", newCtx)
		case "R1", "R2", "R3", "R4", "R5":
			upperLine = strings.TrimPrefix(upperLine, "R")
			number, err := strconv.Atoi(upperLine)
			if err != nil || number <= 0 || number > channelsNum {
				continue
			}
			channelIndex := number - 1
			newChannel := make(chan string)
			inputChannels[channelIndex] = newChannel
			wp.RecoverClosedInputChannel(channelNames[channelIndex], newChannel)
			triggerRecoverChannels[channelIndex] <- newChannel
			fmt.Printf("Recovering Channel '%s'\n", channelNames[channelIndex])
		case "D":
			presentDetails.Store(true)
			fmt.Printf("Presenting receive path on\n")
		case "ND":
			presentDetails.Store(false)
			fmt.Printf("Presenting receive path off\n")
		case "QUIT":
			fmt.Printf("Waiting for all workers to finish...\n")
			wp.Stop()
			<-wp.Done()
			fmt.Printf("Processing finished\n")
			return
		case "WORKERS", "W":
			fmt.Printf("Workers number: %d\n", wp.WorkersNum())
		case "ACTIVE":
			fmt.Printf("Active workers number: %d\n", wp.ActiveWorkersNum())
		}
	}
}

func generateInnerPriorityChannelsContextsAndCancelFuncs() (map[string]context.Context, map[string]context.CancelFunc) {
	priorityChannelsContexts := make(map[string]context.Context)
	priorityChannelsCancelFuncs := make(map[string]context.CancelFunc)

	allPriorityChannels := []string{"Customer A", "Customer B", "Customer Messages", "Urgent Messages"}
	for _, channelName := range allPriorityChannels {
		ctx, cancel := context.WithCancel(context.Background())
		priorityChannelsContexts[channelName] = ctx
		priorityChannelsCancelFuncs[channelName] = cancel
	}
	return priorityChannelsContexts, priorityChannelsCancelFuncs
}

func setAutoDisabledPriorityChannelConfig(config *priority_channels.PriorityChannelConfig) {
	if config == nil {
		return
	}
	config.AutoDisableClosedChannels = true
	for _, channel := range config.Channels {
		if channel.PriorityChannelConfig != nil {
			setAutoDisabledPriorityChannelConfig(channel.PriorityChannelConfig)
		}
	}
}
