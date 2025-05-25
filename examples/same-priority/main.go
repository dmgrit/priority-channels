package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var inputChannels []chan string
	var triggerPauseOrCloseChannels []chan bool

	channelsNum := 8
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseOrCloseChannels = append(triggerPauseOrCloseChannels, make(chan bool))
	}
	var channelsWithPriority = []channels.ChannelWithPriority[string]{
		channels.NewChannelWithPriority("Channel A", inputChannels[0], 5),
		channels.NewChannelWithPriority("Channel B1", inputChannels[1], 4),
		channels.NewChannelWithPriority("Channel B2", inputChannels[2], 4),
		channels.NewChannelWithPriority("Channel B3", inputChannels[3], 4),
		channels.NewChannelWithPriority("Channel C", inputChannels[4], 3),
		channels.NewChannelWithPriority("Channel D1", inputChannels[5], 2),
		channels.NewChannelWithPriority("Channel D2", inputChannels[6], 2),
		channels.NewChannelWithPriority("Channel E", inputChannels[7], 1),
	}

	demoFilePath := filepath.Join(os.TempDir(), "priority_channels_demo.txt")

	fmt.Printf("Same-Priority Demo:\n")
	fmt.Printf("- Press 'A/B1/B2/B3/C/D1/D2/E' to toggle receiving messages from Channels A/B1/B2/B3/C/D1/D2/E\n")
	fmt.Printf("- Press 0 to exit\n\n")
	fmt.Printf("To see the results live, run in another terminal window:\ntail -f %s\n\n", demoFilePath)

	var options []func(*priority_channels.PriorityChannelOptions)
	options = append(options, priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully))
	if len(os.Args) > 1 && os.Args[1] == "-a" {
		options = append(options, priority_channels.AutoDisableClosedChannels())
	}

	ch, err := priority_channels.NewByHighestAlwaysFirst(ctx, channelsWithPriority, options...)
	if err != nil {
		fmt.Printf("Failed to create priority channel: %v\n", err)
		return
	}

	for i := 1; i <= channelsNum; i++ {
		go func(i int) {
			paused := false
			closed := false
			for {
				select {
				case b := <-triggerPauseOrCloseChannels[i-1]:
					if b && !closed {
						close(inputChannels[i-1])
						closed = true
					}
					paused = !paused
				default:
					if !paused && !closed {
						select {
						case b := <-triggerPauseOrCloseChannels[i-1]:
							if b && !closed {
								close(inputChannels[i-1])
								closed = true
							}
							paused = !paused
						case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
						}
					} else {
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}(i)
	}

	go func() {
		f, err := os.Create(demoFilePath)
		if err != nil {
			fmt.Printf("Failed to open file: %v\n", err)
			cancel()
			return
		}
		defer f.Close()
		prevChannel := ""
		streakLength := 0

		for {
			ctx := context.Background()
			_, channel, status := ch.ReceiveWithContext(ctx)
			if status == priority_channels.ReceiveSuccess {
				if channel == prevChannel {
					streakLength++
				} else {
					streakLength = 1
				}
				prevChannel = channel
				logMessage := fmt.Sprintf("%s (%d)\n", channel, streakLength)

				_, err := f.WriteString(logMessage)
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceiveInputChannelClosed {
				_, err := f.WriteString(fmt.Sprintf("Channel '%s' is closed\n", channel))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceiveInnerPriorityChannelClosed {
				_, err = f.WriteString(fmt.Sprintf("Inner Priority Channel '%s' is closed\n", channel))
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
				_, err := f.WriteString(fmt.Sprintf("Unexpected status %s\n", channel))
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

		operation := "Toggle receiving messages from"
		isClose := false
		if upperLine != "C" && strings.HasPrefix(upperLine, "C") {
			isClose = true
			upperLine = strings.TrimPrefix(upperLine, "C")
			operation = "Closing"
		}

		switch upperLine {
		case "A":
			triggerPauseOrCloseChannels[0] <- isClose
			fmt.Printf(operation + " Channel A\n")
		case "B1":
			triggerPauseOrCloseChannels[1] <- isClose
			fmt.Printf(operation + " Channel B1\n")
		case "B2":
			triggerPauseOrCloseChannels[2] <- isClose
			fmt.Printf(operation + " Channel B2\n")
		case "B3":
			triggerPauseOrCloseChannels[3] <- isClose
			fmt.Printf(operation + " Channel B3\n")
		case "C":
			triggerPauseOrCloseChannels[4] <- isClose
			fmt.Printf(operation + " Channel C\n")
		case "D1":
			triggerPauseOrCloseChannels[5] <- isClose
			fmt.Printf(operation + " Channel D1\n")
		case "D2":
			triggerPauseOrCloseChannels[6] <- isClose
			fmt.Printf(operation + " Channel D2\n")
		case "E":
			triggerPauseOrCloseChannels[7] <- isClose
			fmt.Printf(operation + " Channel E\n")
		case "0":
			fmt.Printf("Exiting\n")
			cancel()
			return
		}
	}
}
