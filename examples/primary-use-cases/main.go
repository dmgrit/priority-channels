package main

import (
	"context"
	"fmt"
	"log"
	"os"
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
	var triggerPauseChannels []chan struct{}
	for i := 1; i <= 5; i++ {
		inputChannels = append(inputChannels, make(chan string))
		triggerPauseChannels = append(triggerPauseChannels, make(chan struct{}))
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
			pause := false
			for {
				select {
				case <-triggerPauseChannels[i-1]:
					pause = !pause
				default:
					if !pause {
						select {
						case <-triggerPauseChannels[i-1]:
							pause = !pause
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
			message, channel, ok := ch.Receive()
			if !ok {
				break
			}
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
			time.Sleep(300 * time.Millisecond)
		}
	}()

	for {
		var number int
		_, err := fmt.Scanf("%d", &number)
		if err != nil || number < 0 || number > 5 {
			continue
		}
		if number == 0 {
			fmt.Printf("Exiting\n")
			cancel()
			break
		}
		fmt.Printf("Toggling pause/resume for Channel %d\n", number)
		triggerPauseChannels[number-1] <- struct{}{}
	}
}
