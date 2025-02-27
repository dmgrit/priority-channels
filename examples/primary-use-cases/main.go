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
	"github.com/dmgrit/priority-channels/strategies"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	var channelsWithFreqRatio []channels.ChannelWithFreqRatio[string]
	var channelsWithPriority []channels.ChannelWithPriority[string]
	var channelsWithProbability []channels.ChannelWithWeight[string, float64]
	var inputChannels []chan string
	var triggerPauseOrCloseChannels []chan bool
	var pauseChannels []chan struct{}
	var resumeChannels []chan struct{}

	channelsNum := 5
	arithmeticSum := 0
	for i := 1; i <= channelsNum; i++ {
		arithmeticSum += i
	}
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseOrCloseChannels = append(triggerPauseOrCloseChannels, make(chan bool))
		pauseChannels = append(pauseChannels, make(chan struct{}))
		resumeChannels = append(resumeChannels, make(chan struct{}))
		channelsWithFreqRatio = append(channelsWithFreqRatio, channels.NewChannelWithFreqRatio(
			fmt.Sprintf("Freq Ratio %d", i),
			inputChannels[i-1],
			i))
		channelsWithPriority = append(channelsWithPriority, channels.NewChannelWithPriority(
			fmt.Sprintf("Priority %d", i),
			inputChannels[i-1],
			i))
		probability := float64(i) / float64(arithmeticSum)
		channelsWithProbability = append(channelsWithProbability, channels.NewChannelWithWeight[string, float64](
			fmt.Sprintf("Probability %.2f", probability),
			inputChannels[i-1],
			probability))
	}

	fmt.Printf("Select Demo:\n")
	fmt.Printf("- Priority Channel by Frequency Ratio - (F)\n")
	fmt.Printf("- Priority Channel by Highest Priority Always First - (H)\n")
	fmt.Printf("- Priority Channel by Probabilities - (P)\n")
	fmt.Printf("- Press F, H or P to continue\n")

	isByFrequencyRatio := false
	isByHighestAlwaysFirst := false
	isByProbability := false
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
			isByHighestAlwaysFirst = true
			break
		} else if strings.ToUpper(option) == "P" {
			isByProbability = true
			break
		}
	}

	var ch *priority_channels.PriorityChannel[string]
	var err error

	if isByFrequencyRatio {
		ch, err = priority_channels.NewByFrequencyRatio(ctx, channelsWithFreqRatio)
	} else if isByHighestAlwaysFirst {
		ch, err = priority_channels.NewByHighestAlwaysFirst(ctx, channelsWithPriority)
	} else {
		ch, err = priority_channels.NewByStrategy(ctx, strategies.NewByProbability(), channelsWithProbability)
	}
	if err != nil {
		fmt.Printf("Failed to create priority channel: %v\n", err)
		return
	}
	fullSpeed := isByProbability

	for i := 1; i <= channelsNum; i++ {
		go func(i int) {
			paused := false
			resumeEnabled := false
			closed := false
			for {
				select {
				case <-pauseChannels[i-1]:
					if !paused {
						paused = true
						resumeEnabled = true
					}
				case <-resumeChannels[i-1]:
					if resumeEnabled {
						paused = false
						resumeEnabled = false
					}
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
						if !fullSpeed {
							time.Sleep(100 * time.Millisecond)
						} else {
							time.Sleep(1 * time.Millisecond)
						}
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
		totalCount := 0
		countPerChannel := make(map[string]int)
		lastAppearancesPerChannel := make(map[string][]bool)
		if isByProbability {
			for _, c := range channelsWithProbability {
				countPerChannel[c.ChannelName()] = 0
				lastAppearancesPerChannel[c.ChannelName()] = make([]bool, 0, 1000)
			}
		}

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
				logMessage := fmt.Sprintf("%s (%d)\n", message, streakLength)

				if isByProbability {
					if totalCount < 1000 {
						totalCount++
					}
					for channelName, lapc := range lastAppearancesPerChannel {
						currentSelected := channel == channelName
						lastAppearancesPerChannel[channelName] = append(lastAppearancesPerChannel[channelName], currentSelected)
						if len(lastAppearancesPerChannel[channelName]) > 1000 {
							if !lapc[0] && currentSelected {
								countPerChannel[channelName] = countPerChannel[channelName] + 1
							} else if lapc[0] && !currentSelected {
								countPerChannel[channelName] = countPerChannel[channelName] - 1
							}
							lastAppearancesPerChannel[channelName] = lastAppearancesPerChannel[channelName][1:]
						} else if currentSelected {
							countPerChannel[channel] = countPerChannel[channel] + 1
						}
					}
					logMessage = fmt.Sprintf("%s (%.2f)\n", message, float64(countPerChannel[channel])/float64(totalCount))
				}
				_, err := f.WriteString(logMessage)
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
			if !fullSpeed {
				time.Sleep(300 * time.Millisecond)
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		var number int
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if fullSpeed {
			if strings.ToUpper(line) == "P" {
				for i := 0; i < channelsNum; i++ {
					pauseChannels[i] <- struct{}{}
				}
				continue
			} else if strings.ToUpper(line) == "R" {
				for i := 0; i < channelsNum; i++ {
					resumeChannels[i] <- struct{}{}
				}
			}
		}

		isClose := false
		if strings.HasPrefix(line, "c") {
			isClose = true
			line = strings.TrimPrefix(line, "c")
		}
		number, err := strconv.Atoi(line)
		if err != nil || number < 0 || number > channelsNum {
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
