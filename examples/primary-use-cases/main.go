package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var channelsWithFreqRatio []channels.ChannelWithFreqRatio[string]
	var channelsWithPriority []channels.ChannelWithPriority[string]
	var inputChannels []chan string
	var triggerPauseOrCloseChannels []chan bool
	for i := 1; i <= 5; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseOrCloseChannels = append(triggerPauseOrCloseChannels, make(chan bool))
		channelsWithFreqRatio = append(channelsWithFreqRatio, channels.NewChannelWithFreqRatio(
			fmt.Sprintf("Freq Ratio %d", i),
			inputChannels[i-1],
			i))
		channelsWithPriority = append(channelsWithPriority, channels.NewChannelWithPriority(
			fmt.Sprintf("Priority %d", i),
			inputChannels[i-1],
			i))
	}

	fmt.Printf("Select Demo:\n")
	fmt.Printf("- Priority Channel by Frequency Ratio - (F)\n")
	fmt.Printf("- Priority Channel by Highest Priority Always First - (H)\n")
	fmt.Printf("- Press F or H to continue\n")

	isByFrequencyRatio := true
	for {
		var option string
		_, err := fmt.Scanf("%s", &option)
		if err != nil {
			log.Fatal(err)
		}
		if strings.ToUpper(option) == "F" {
			isByFrequencyRatio = true
			break
		} else if strings.ToUpper(option) == "H" {
			isByFrequencyRatio = false
			break
		}
	}

	var ch *priority_channels.PriorityChannel[string]
	var err error
	if isByFrequencyRatio {
		ch, err = priority_channels.NewByFrequencyRatio(ctx, channelsWithFreqRatio)
	} else {
		ch, err = priority_channels.NewByHighestAlwaysFirst(ctx, channelsWithPriority)
	}
	if err != nil {
		fmt.Printf("Failed to create priority channel: %v\n", err)
		return
	}

	for i := 1; i <= 5; i++ {
		go func(i int) {
			paused := false
			closed := false
			for {
				select {
				case b := <-triggerPauseOrCloseChannels[i-1]:
					if b {
						close(inputChannels[i-1])
						closed = true
					}
					paused = !paused
				default:
					if !paused && !closed {
						select {
						case b := <-triggerPauseOrCloseChannels[i-1]:
							if b {
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
		f, err := os.Create("/tmp/priority_channels_demo.txt")
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
			message, channel, status := ch.ReceiveWithContext(ctx)
			if status == priority_channels.ReceiveSuccess {
				if channel == prevChannel {
					streakLength++
				} else {
					streakLength = 1
				}
				prevChannel = channel
				_, err := f.WriteString(fmt.Sprintf("%s (%d)\n", message, streakLength))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceiveChannelClosed {
				_, err := f.WriteString(fmt.Sprintf("Channel '%s' is closed\n", channel))
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
					cancel()
					break
				}
			} else if status == priority_channels.ReceivePriorityChannelCancelled {
				var err error
				if channel == "" {
					_, err = f.WriteString(fmt.Sprintf("Priority Channel is closed\n"))
				} else {
					_, err = f.WriteString(fmt.Sprintf("Priority Channel '%s' is closed\n", channel))
				}
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
				status != priority_channels.ReceiveChannelClosed &&
				(status != priority_channels.ReceivePriorityChannelCancelled || channel == "") {
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
		var number int
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		isClose := false
		if strings.HasPrefix(line, "c") {
			isClose = true
			line = strings.TrimPrefix(line, "c")
		}
		number, err := strconv.Atoi(line)
		if err != nil || number < 0 || number > 5 {
			continue
		}
		if number == 0 {
			fmt.Printf("Exiting\n")
			cancel()
			break
		}
		if isClose {
			fmt.Printf("Closing Channel %d\n", number)
		} else {
			fmt.Printf("Toggling pause/resume for Channel %d\n", number)
		}
		triggerPauseOrCloseChannels[number-1] <- isClose
	}
}
