package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	var triggerPauseOrCloseChannels []chan bool
	var triggerRecoverChannels []chan chan string

	channelsNum := 8
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseOrCloseChannels = append(triggerPauseOrCloseChannels, make(chan bool))
		triggerRecoverChannels = append(triggerRecoverChannels, make(chan chan string))
	}

	channelsOrder := map[string]int{
		"Channel A": 1,
		"Channel B": 2,
		"Channel C": 3,
	}

	priorityConfig := priority_channels.Configuration{
		PriorityChannel: &priority_channels.PriorityChannelConfig{
			Method:                    priority_channels.ByFrequencyRatioMethodConfig,
			AutoDisableClosedChannels: true,
			FrequencyMethod:           priority_channels.ProbabilisticByMultipleRandCallsFrequencyMethodConfig,
			Channels: []priority_channels.ChannelConfig{
				{Name: "Channel A", FreqRatio: 6},
				{Name: "Channel B", FreqRatio: 3},
				{Name: "Channel C", FreqRatio: 1},
			},
		},
	}

	channelNameToChannel := map[string]<-chan string{
		"Channel A": inputChannels[0],
		"Channel B": inputChannels[1],
		"Channel C": inputChannels[2],
	}

	for i := 1; i <= channelsNum; i++ {
		go func(triggerPauseOrCloseChannel chan bool, inputChannel chan string, triggerRecoverChannel chan chan string) {
			paused := false
			closed := false
			j := 0
			for {
				j++
				imageName := randString(16)
				select {
				case b := <-triggerPauseOrCloseChannel:
					if b && !closed {
						close(inputChannel)
						closed = true
					} else if !closed {
						paused = !paused
					}
				case inputChannel = <-triggerRecoverChannel:
					closed = false
				default:
					if !paused && !closed {
						select {
						case b := <-triggerPauseOrCloseChannel:
							if b && !closed {
								close(inputChannel)
								closed = true
							} else {
								paused = !paused
							}
						case inputChannel <- fmt.Sprintf("image-%s-%d", imageName, j):
						}
					} else {
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}(triggerPauseOrCloseChannels[i-1], inputChannels[i-1], triggerRecoverChannels[i-1])
	}

	receivedMsgs := 0
	byChannelName := make(map[string]int)
	var receivedMsgsMutex sync.Mutex

	processFn := func(d priority_channels.Delivery[string]) {
		time.Sleep(100 * time.Millisecond)
		receivedMsgsMutex.Lock()
		receivedMsgs++
		channelName := d.ReceiveDetails.ChannelName
		byChannelName[channelName] = byChannelName[channelName] + 1
		receivedMsgsMutex.Unlock()
	}
	closureBehavior := priority_channels.ClosureBehavior{
		InputChannelClosureBehavior:    priority_channels.PauseOnClosed,
		PriorityChannelClosureBehavior: priority_channels.PauseOnClosed,
		NoOpenChannelsBehavior:         priority_channels.PauseWhenNoOpenChannels,
	}
	wp, err := priority_channels.NewDynamicPriorityProcessor(ctx, channelNameToChannel, priorityConfig, 3, closureBehavior)
	if err != nil {
		fmt.Printf("failed to initialize dynamic priority processor: %v\n", err)
		os.Exit(1)
	}

	notifyCloseCh := make(chan priority_channels.ClosedChannelEvent[string])
	closedChannelNameToRecoveryFn := make(map[string]struct{})
	var closedChannelNameToRecoveryFnMtx sync.Mutex
	go func() {
		for closedEvent := range notifyCloseCh {
			closedChannelNameToRecoveryFnMtx.Lock()
			closedChannelNameToRecoveryFn[closedEvent.ChannelName] = struct{}{}
			closedChannelNameToRecoveryFnMtx.Unlock()
		}
	}()
	wp.NotifyClose(notifyCloseCh)

	err = wp.Process(processFn)
	if err != nil {
		fmt.Printf("failed to start processing messages: %v\n", err)
		os.Exit(1)
	}

	demoFilePath := filepath.Join(os.TempDir(), "dynamic_processor_demo.txt")
	f, err := os.Create(demoFilePath)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		cancel()
		return
	}
	defer f.Close()

	fmt.Printf("Dynamic-Priority-Processor Demo:\n")
	fmt.Printf("- Press 'a/b/c' to toggle receiving messages from Channels A/B/C\n")
	fmt.Printf("- Press 'w <workers_num>' to update number of workers in the worker pool\n")
	fmt.Printf("- Press 'l <priority_configuration_file>' to load a different priority configuration\n")
	fmt.Printf("- Press 'w' or 'workers' to see the number of worker goroutines configured in the worker pool\n")
	fmt.Printf("- Press 'active' to see the number of goroutines currently processing messages\n")
	fmt.Printf("- Press quit to exit\n\n")
	fmt.Printf("To see the results live, run in another terminal window:\ntail -f %s\n\n", demoFilePath)

	tickerCh := time.Tick(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tickerCh:
				receivedMsgsMutex.Lock()
				_, _ = f.WriteString(strings.Repeat("=", 80) + "\n")
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
					stopped, reason, channelName := wp.Status()
					if stopped {
						switch reason {
						case priority_channels.UnknownExitReason:
							_, _ = f.WriteString("Worker pool stopped: Unknown reason\n")
						case priority_channels.ChannelClosed:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool stopped: Channel '%s' closed\n", channelName))
						case priority_channels.PriorityChannelClosed:
							_, _ = f.WriteString(fmt.Sprintf("Worker pool stopped: Priority Channel '%s' closed\n", channelName))
						case priority_channels.NoOpenChannels:
							_, _ = f.WriteString("Worker pool stopped: No open channels\n")
						case priority_channels.ContextCanceled:
							_, _ = f.WriteString("Worker pool stopped: Context Canceled\n")
						}
					}
				}
				_, _ = f.WriteString(strings.Repeat("=", 80) + "\n")

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

		words := strings.Split(line, " ")
		if len(words) == 1 {
			switch words[0] {
			case "a", "b", "c", "ca", "cb", "cc":
				// toggle or close channels
				isClose := words[0] != "c" && strings.HasPrefix(words[0], "c")
				operation := "Toggle receiving messages from"
				if isClose {
					operation = "Closing"
					words[0] = strings.TrimPrefix(words[0], "c")
				}
				switch words[0] {
				case "a":
					triggerPauseOrCloseChannels[0] <- isClose
					fmt.Printf(operation + " Channel A\n")
				case "b":
					triggerPauseOrCloseChannels[1] <- isClose
					fmt.Printf(operation + " Channel B\n")
				case "c":
					triggerPauseOrCloseChannels[2] <- isClose
					fmt.Printf(operation + " Channel C\n")
				}
			case "ra", "rb", "rc":
				var channelIndex int
				var channelName string
				switch words[0] {
				case "ra":
					channelIndex = 0
					channelName = "Channel A"
				case "rb":
					channelIndex = 1
					channelName = "Channel B"
				case "rc":
					channelIndex = 2
					channelName = "Channel C"
				}
				closedChannelNameToRecoveryFnMtx.Lock()
				_, ok := closedChannelNameToRecoveryFn[channelName]
				if !ok {
					closedChannelNameToRecoveryFnMtx.Unlock()
					fmt.Printf("Recovery is not enabled for channel %s\n", channelName)
					continue
				}
				newChannel := make(chan string)
				inputChannels[channelIndex] = newChannel
				wp.RecoverClosedInputChannel(channelName, newChannel)
				delete(closedChannelNameToRecoveryFn, channelName)
				closedChannelNameToRecoveryFnMtx.Unlock()
				triggerRecoverChannels[channelIndex] <- newChannel
				fmt.Printf("Recovering %s\n", channelName)
			case "quit":
				wp.Stop()
				fmt.Printf("Waiting for all workers to finish...\n")
				<-wp.Done()
				fmt.Printf("Processing finished\n")
				return
			case "workers", "w":
				fmt.Printf("Workers number: %d\n", wp.WorkersNum())
			case "active":
				fmt.Printf("Active workers number: %d\n", wp.ActiveWorkersNum())
			}
			continue
		}

		if len(words) != 2 {
			continue
		}

		switch words[0] {
		case "load", "l":
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

			if err := wp.UpdatePriorityConfiguration(priorityConfig); err != nil {
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
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
