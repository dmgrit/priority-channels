package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	msgsChannels := make([]chan string, 2)
	msgsChannels[0] = make(chan string)
	msgsChannels[1] = make(chan string)

	strategiesByName := map[string]strategies.DynamicSubStrategy{
		"Regular":              frequency_strategies.NewWithStrictOrderFully(),
		"A-Reserved":           frequency_strategies.NewWithStrictOrderFully(),
		"A-Reserved-Exclusive": priority_strategies.NewByHighestAlwaysFirst(),
		"B-Reserved":           frequency_strategies.NewWithStrictOrderFully(),
		"B-Reserved-Exclusive": priority_strategies.NewByHighestAlwaysFirst(),
	}
	channelsWithWeights := []channels.ChannelWithWeight[string, map[string]interface{}]{
		channels.NewChannelWithWeight("Channel A", msgsChannels[0],
			map[string]interface{}{
				"Regular":              1,
				"A-Reserved":           5,
				"A-Reserved-Exclusive": 2,
				"B-Reserved":           1,
				"B-Reserved-Exclusive": 1,
			}),
		channels.NewChannelWithWeight("Channel B", msgsChannels[1],
			map[string]interface{}{
				"Regular":              1,
				"A-Reserved":           1,
				"A-Reserved-Exclusive": 1,
				"B-Reserved":           5,
				"B-Reserved-Exclusive": 2,
			}),
	}

	var mode string
	var modeMutex sync.RWMutex

	currentStrategySelector := func() string {
		modeMutex.RLock()
		defer modeMutex.RUnlock()

		switch mode {
		case "A":
			return "A-Reserved"
		case "AE":
			return "A-Reserved-Exclusive"
		case "B":
			return "B-Reserved"
		case "BE":
			return "B-Reserved-Exclusive"
		default:
			return "Regular"
		}
	}

	setOrToggleCurrentMode := func(newMode string) {
		modeMutex.Lock()
		defer modeMutex.Unlock()

		if mode == newMode {
			mode = ""
		} else {
			mode = newMode
		}
	}

	fmt.Printf("Dynamic Strategy Demo:\n")
	fmt.Printf("- Press 'A' to toggle 'Customer A' reserved time mode\n")
	fmt.Printf("- Press 'AE' to toggle 'Customer A' reserved exclusive time mode\n")
	fmt.Printf("- Press 'B' to toggle 'Customer B' reserved time mode\n")
	fmt.Printf("- Press 'BE' to toggle 'Customer A' reserved exclusive time mode\n")
	fmt.Printf("- Press 0 to exit\n\n")

	ch, err := priority_channels.NewByStrategy(ctx,
		strategies.NewDynamic(strategiesByName, currentStrategySelector),
		channelsWithWeights,
	)
	if err != nil {
		fmt.Printf("Failed to create priority channel: %v\n", err)
		return
	}

	for i := 0; i <= 1; i++ {
		go func(i int) {
			var customer string
			if i == 0 {
				customer = "Customer A"
			} else {
				customer = "Customer B"
			}
			for {
				select {
				case <-ctx.Done():
					return
				case msgsChannels[i] <- customer:
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
				_, err := f.WriteString("Exiting\n")
				if err != nil {
					fmt.Printf("Failed to write to file: %v\n", err)
				}
				cancel()
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

	reader := bufio.NewReader(os.Stdin)
	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		setOrToggleCurrentMode(line)

		if line == "0" {
			fmt.Printf("Exiting\n")
			cancel()
			break
		}
	}
}
