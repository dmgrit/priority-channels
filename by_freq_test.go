package priority_channels_test

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pc "github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

type Msg struct {
	Body string
}

func TestProcessMessagesByFrequencyRatio(t *testing.T) {
	msgsChannels := make([]chan *Msg, 4)
	msgsChannels[0] = make(chan *Msg, 15)
	msgsChannels[1] = make(chan *Msg, 15)
	msgsChannels[2] = make(chan *Msg, 15)
	msgsChannels[3] = make(chan *Msg, 15)

	channels := []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithFreqRatio("Priority-5", msgsChannels[1], 5),
		channels.NewChannelWithFreqRatio("Priority-10", msgsChannels[2], 10),
		channels.NewChannelWithFreqRatio("Priority-1000", msgsChannels[3], 1000),
	}

	for i := 0; i <= 2; i++ {
		for j := 1; j <= 15; j++ {
			msgsChannels[i] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[i].ChannelName(), j)}
		}
	}
	msgsChannels[3] <- &Msg{Body: "Priority-1000 Msg-1"}

	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		results = append(results, msg)
	}
	ctx, cancel := context.WithCancel(context.Background())

	priorityChannel, err := pc.NewByFrequencyRatio(ctx, channels, pc.WithFrequencyMethod(pc.StrictOrderFully))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	go pc.ProcessPriorityChannelMessages(priorityChannel, msgProcessor)

	time.Sleep(3 * time.Second)
	cancel()

	expectedResults := []*Msg{
		{Body: "Priority-1000 Msg-1"},
		{Body: "Priority-10 Msg-1"},
		{Body: "Priority-10 Msg-2"},
		{Body: "Priority-10 Msg-3"},
		{Body: "Priority-10 Msg-4"},
		{Body: "Priority-10 Msg-5"},
		{Body: "Priority-10 Msg-6"},
		{Body: "Priority-10 Msg-7"},
		{Body: "Priority-10 Msg-8"},
		{Body: "Priority-10 Msg-9"},
		{Body: "Priority-10 Msg-10"},
		{Body: "Priority-5 Msg-1"},
		{Body: "Priority-5 Msg-2"},
		{Body: "Priority-5 Msg-3"},
		{Body: "Priority-5 Msg-4"},
		{Body: "Priority-5 Msg-5"},
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-10 Msg-11"},
		{Body: "Priority-10 Msg-12"},
		{Body: "Priority-10 Msg-13"},
		{Body: "Priority-10 Msg-14"},
		{Body: "Priority-10 Msg-15"},
		{Body: "Priority-5 Msg-6"},
		{Body: "Priority-5 Msg-7"},
		{Body: "Priority-5 Msg-8"},
		{Body: "Priority-5 Msg-9"},
		{Body: "Priority-5 Msg-10"},
		{Body: "Priority-1 Msg-2"},
		{Body: "Priority-5 Msg-11"},
		{Body: "Priority-5 Msg-12"},
		{Body: "Priority-5 Msg-13"},
		{Body: "Priority-5 Msg-14"},
		{Body: "Priority-5 Msg-15"},
		{Body: "Priority-1 Msg-3"},
		{Body: "Priority-1 Msg-4"},
		{Body: "Priority-1 Msg-5"},
		{Body: "Priority-1 Msg-6"},
		{Body: "Priority-1 Msg-7"},
		{Body: "Priority-1 Msg-8"},
		{Body: "Priority-1 Msg-9"},
		{Body: "Priority-1 Msg-10"},
		{Body: "Priority-1 Msg-11"},
		{Body: "Priority-1 Msg-12"},
		{Body: "Priority-1 Msg-13"},
		{Body: "Priority-1 Msg-14"},
		{Body: "Priority-1 Msg-15"},
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, but got %d", len(expectedResults), len(results))
	}
	for i := range results {
		if results[i].Body != expectedResults[i].Body {
			t.Errorf("Result %d: Expected message %s, but got %s",
				i, expectedResults[i].Body, results[i].Body)
		}
	}
}

func TestErrorOnInvalidFrequencyMethod(t *testing.T) {
	ctx := context.Background()
	channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio("Channel A", make(chan string), 1),
		channels.NewChannelWithFreqRatio("Channel B", make(chan string), 2),
		channels.NewChannelWithFreqRatio("Channel C", make(chan string), 3),
	}

	_, err := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio, pc.WithFrequencyMethod(22))
	if err == nil {
		t.Fatalf("Expected invalid frequency method error but got none")
	}
	if err.Error() != pc.ErrInvalidFrequencyMethod.Error() {
		t.Fatalf("Expected invalid frequency method but got %v", err)
	}
}

func TestProcessMessagesByFrequencyRatio_TenThousandMessages(t *testing.T) {
	var testCases = []struct {
		Name            string
		FreqRatioMethod pc.FrequencyMethod
	}{
		{
			Name:            "StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
		},
		{
			Name:            "StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
		},
		{
			Name:            "ProbabilisticWithCasesDuplications",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
		},
		{
			Name:            "ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testProcessMessagesByFrequencyRatioWithMethod(t, tc.FreqRatioMethod, 10000)
		})
	}
}

func testProcessMessagesByFrequencyRatioWithMethod(t *testing.T, freqRatioMethod pc.FrequencyMethod, messagesNum int) {
	ctx, cancel := context.WithCancel(context.Background())
	var inputChannels []chan string

	channelsNum := 5
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
	}

	channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio("Channel A", inputChannels[0], 1),
		channels.NewChannelWithFreqRatio("Channel B", inputChannels[1], 2),
		channels.NewChannelWithFreqRatio("Channel C", inputChannels[2], 3),
		channels.NewChannelWithFreqRatio("Channel D", inputChannels[3], 4),
		channels.NewChannelWithFreqRatio("Channel E", inputChannels[4], 5),
	}

	freqTotalSum := 0.0
	for i := 1; i <= channelsNum; i++ {
		freqTotalSum += float64(channelsWithFreqRatio[i-1].FreqRatio())
	}
	expectedRatios := make(map[string]float64)
	for _, ch := range channelsWithFreqRatio {
		expectedRatios[ch.ChannelName()] = float64(ch.FreqRatio()) / freqTotalSum
	}

	for i := 1; i <= channelsNum; i++ {
		go func(i int) {
			for j := 1; j <= messagesNum; j++ {
				select {
				case <-ctx.Done():
					return
				case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
				}
			}
		}(i)
	}

	ch, err := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio, pc.WithFrequencyMethod(freqRatioMethod))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	totalCount := 0
	countPerChannel := make(map[string]int)
	go func() {
		for {
			_, channel, ok := ch.Receive()
			if !ok {
				return
			}
			time.Sleep(1 * time.Microsecond)
			totalCount++
			countPerChannel[channel] = countPerChannel[channel] + 1
			if totalCount == messagesNum {
				cancel()
				return
			}
		}
	}()

	<-ctx.Done()

	for _, channel := range channelsWithFreqRatio {
		expectedRatio := expectedRatios[channel.ChannelName()]
		actualRatio := float64(countPerChannel[channel.ChannelName()]) / float64(totalCount)
		if math.Abs(expectedRatio-actualRatio) > 0.03 {
			t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
				channel.ChannelName(), expectedRatio, actualRatio)
		}
		t.Logf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
			channel.ChannelName(), expectedRatio, actualRatio)
	}
}

func TestProcessMessagesByFrequencyRatio_RandomChannelsList(t *testing.T) {
	var testCases = []struct {
		Name            string
		FreqRatioMethod pc.FrequencyMethod
	}{
		{
			Name:            "StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
		},
		{
			Name:            "StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
		},
		{
			Name:            "ProbabilisticWithCasesDuplications",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
		},
		{
			Name:            "ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
		},
	}

	channelsWithFreqRatio, channelsWithExpectedRatios := generateRandomFreqRatioList(8, 16)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testProcessMessagesByFrequencyRatio_RandomChannelsListWithMethod(t,
				channelsWithFreqRatio, channelsWithExpectedRatios, nil, tc.FreqRatioMethod, false)
		})
	}
}

func TestProcessMessagesByFrequencyRatio_RandomChannelsListSubset(t *testing.T) {
	var testCases = []struct {
		Name            string
		FreqRatioMethod pc.FrequencyMethod
		CloseChannels   bool
	}{
		{
			Name:            "StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
		},
		{
			Name:            "StrictOrderAcrossCycles-CloseChannels",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
			CloseChannels:   true,
		},
		{
			Name:            "StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
		},
		{
			Name:            "StrictOrderFully-CloseChannels",
			FreqRatioMethod: pc.StrictOrderFully,
			CloseChannels:   true,
		},
		{
			Name:            "ProbabilisticWithCasesDuplications",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
		},
		{
			Name:            "ProbabilisticWithCasesDuplications-CloseChannels",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
			CloseChannels:   true,
		},
		{
			Name:            "ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
		},
		{
			Name:            "ProbabilisticByMultipleRandCalls-CloseChannels",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
			CloseChannels:   true,
		},
	}

	channelsWithFreqRatio, channelsWithExpectedRatios := generateRandomFreqRatioList(8, 16)
	channelIndexesSubset := getRandomChannelIndexesSubset(len(channelsWithExpectedRatios))
	if channelIndexesSubset != nil {
		recomputeRandomFreqListExpectedRatios(channelsWithFreqRatio, channelsWithExpectedRatios, channelIndexesSubset)
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testProcessMessagesByFrequencyRatio_RandomChannelsListWithMethod(t,
				channelsWithFreqRatio, channelsWithExpectedRatios, channelIndexesSubset,
				tc.FreqRatioMethod, tc.CloseChannels)
		})
	}
}

func generateRandomFreqRatioList(minListSize, maxListSize int) ([]channels.ChannelWithFreqRatio[string], map[string]*channelWithExpectedRatio) {
	childrenNum := rand.N(maxListSize-minListSize+1) + minListSize
	totalSum := 0.0
	weights := make([]int, 0, childrenNum)
	for i := 0; i < childrenNum; i++ {
		w := rand.N(10) + 1
		weights = append(weights, w)
		totalSum += float64(w)
	}
	expectedRatios := make([]float64, 0, childrenNum)
	accExpectedRatio := 0.0
	for i := 0; i < childrenNum; i++ {
		if i == childrenNum-1 {
			expectedRatios = append(expectedRatios, 1.0-accExpectedRatio)
			break
		}
		expectedRatio := float64(weights[i]) / totalSum
		expectedRatios = append(expectedRatios, expectedRatio)
		accExpectedRatio += expectedRatio
	}
	channelsWithExpectedRatios := make(map[string]*channelWithExpectedRatio)
	channelsWithFreqRatio := make([]channels.ChannelWithFreqRatio[string], 0, childrenNum)
	for i := 0; i < childrenNum; i++ {
		cwr := &channelWithExpectedRatio{
			channel:       make(chan string, 10),
			expectedRatio: expectedRatios[i],
			channelIndex:  i,
		}
		channelName := fmt.Sprintf("channel-%d", i)
		channelsWithExpectedRatios[channelName] = cwr
		childChannel := channels.NewChannelWithFreqRatio(channelName, cwr.channel, weights[i])
		channelsWithFreqRatio = append(channelsWithFreqRatio, childChannel)
	}
	return channelsWithFreqRatio, channelsWithExpectedRatios
}

func recomputeRandomFreqListExpectedRatios(
	channelsWithFreqRatio []channels.ChannelWithFreqRatio[string],
	channelsWithExpectedRatios map[string]*channelWithExpectedRatio,
	channelIndexesSubset map[int]struct{}) {
	totalSum := 0.0
	for i := range channelIndexesSubset {
		totalSum += float64(channelsWithFreqRatio[i].FreqRatio())
	}
	for i := 0; i < len(channelsWithFreqRatio); i++ {
		c := channelsWithFreqRatio[i]
		cwr := channelsWithExpectedRatios[c.ChannelName()]
		if _, ok := channelIndexesSubset[i]; !ok {
			cwr.expectedRatio = 0.0
			continue
		}
		expectedRatio := float64(c.FreqRatio()) / totalSum
		cwr.expectedRatio = expectedRatio
	}
}

func testProcessMessagesByFrequencyRatio_RandomChannelsListWithMethod(t *testing.T,
	channelsWithFreqRatio []channels.ChannelWithFreqRatio[string],
	channelsWithExpectedRatios map[string]*channelWithExpectedRatio,
	channelIndexesSubset map[int]struct{},
	frequencyMethod pc.FrequencyMethod,
	closeChannels bool) {
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := pc.NewByFrequencyRatio[string](ctx, channelsWithFreqRatio, pc.WithFrequencyMethod(frequencyMethod),
		pc.AutoDisableClosedChannels())
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	if len(channelIndexesSubset) > 0 {
		t.Logf("Taking subset of %d channels out of %d\n", len(channelIndexesSubset), len(channelsWithFreqRatio))
		for channelName, cwr := range channelsWithExpectedRatios {
			if _, ok := channelIndexesSubset[cwr.channelIndex]; !ok {
				delete(channelsWithExpectedRatios, channelName)
				if closeChannels {
					close(cwr.channel)
				}
			}
		}
	}

	messagesNum := 100000
	for channelName, cwr := range channelsWithExpectedRatios {
		go func(channelName string, cwr *channelWithExpectedRatio) {
			for j := 1; j <= messagesNum; j++ {
				select {
				case <-ctx.Done():
					return
				case cwr.channel <- fmt.Sprintf("Message %d", j):
				}
			}
		}(channelName, cwr)
	}

	totalCount := 0
	countPerChannel := make(map[string]int)
	go func() {
		for {
			_, channel, ok := ch.Receive()
			if !ok {
				return
			}
			time.Sleep(1 * time.Microsecond)
			totalCount++
			countPerChannel[channel] = countPerChannel[channel] + 1
			if totalCount == messagesNum {
				cancel()
				return
			}
		}
	}()

	<-ctx.Done()

	totalDiff := 0.0
	channelNames := make([]string, 0, len(channelsWithExpectedRatios))
	for channelName := range channelsWithExpectedRatios {
		channelNames = append(channelNames, channelName)
	}
	sort.Strings(channelNames)

	for _, channelName := range channelNames {
		cwr := channelsWithExpectedRatios[channelName]
		actualRatio := float64(countPerChannel[channelName]) / float64(totalCount)
		diff := math.Abs(cwr.expectedRatio - actualRatio)
		diffPercentage := (diff / cwr.expectedRatio) * 100
		var diffThreshold float64
		switch frequencyMethod {
		case pc.StrictOrderAcrossCycles, pc.StrictOrderFully:
			diffThreshold = 3
		case pc.ProbabilisticByCaseDuplication:
			diffThreshold = 8
		case pc.ProbabilisticByMultipleRandCalls:
			diffThreshold = 5
		}
		if diffPercentage > diffThreshold && diff > 0.001 {
			t.Errorf("Unexpected Ratio: Channel %s: expected messages number by ratio %.5f, got %.5f (%.1f%%)",
				channelName, cwr.expectedRatio, actualRatio, diffPercentage)
		} else {
			t.Logf("Channel %s: expected messages number by ratio %.5f, got %.5f (%.1f%%)",
				channelName, cwr.expectedRatio, actualRatio, diffPercentage)
		}
		totalDiff += diff
	}
	t.Logf("Total diff: %.5f\n", totalDiff)
}

func TestProcessMessagesByFrequencyRatioWithGoroutines(t *testing.T) {
	testCases := []struct {
		Name     string
		isSubset bool
	}{
		{
			Name:     "Full list",
			isSubset: false,
		},
		{
			Name:     "Subset",
			isSubset: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(*testing.T) {
			channel1 := make(chan string, 10)
			channel3 := make(chan string, 10)
			channel5 := make(chan string, 10)
			channel10 := make(chan string, 10)

			channelsWithFreqRatios := []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio("Channel 1", channel1, 1),
				channels.NewChannelWithFreqRatio("Channel 3", channel3, 3),
				channels.NewChannelWithFreqRatio("Channel 5", channel5, 5),
				channels.NewChannelWithFreqRatio("Channel 10", channel10, 10),
			}

			allChannels := []chan string{channel1, channel3, channel5, channel10}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			channelNameToIndex := map[string]int{
				"Channel 1":  0,
				"Channel 3":  1,
				"Channel 5":  2,
				"Channel 10": 3,
			}
			m := make(map[int]int)
			var mtx sync.Mutex

			var wg sync.WaitGroup
			// senders
			for i := 0; i < 4; i++ {
				wg.Add(1)
				go func(c chan string, i int) {
					if tc.isSubset && (i == 0 || i == 1) {
						wg.Done()
						return
					}
					defer wg.Done()
					for {
						select {
						case <-ctx.Done():
							return
						case c <- fmt.Sprintf("channel %d", i):
						}
					}
				}(allChannels[i], i)
			}

			onMessageReceived := func(msg string, channelName string) {
				mtx.Lock()
				idx := channelNameToIndex[channelName]
				m[idx]++
				mtx.Unlock()
				// simulate some activity
				time.Sleep(50 * time.Microsecond)
			}
			processingContextCanceled := make(chan struct{})
			onProcessingFinished := func(reason pc.ExitReason) {
				if reason == pc.ContextCancelled {
					processingContextCanceled <- struct{}{}
				} else {
					t.Errorf("Unexpected processing finished reason: %v", reason)
				}
			}

			err := pc.ProcessByFrequencyRatioWithGoroutines(ctx, channelsWithFreqRatios, onMessageReceived, nil, onProcessingFinished)
			if err != nil {
				t.Fatalf("Unexpected error on calling process by frequency ratio with goroutines: %v", err)
			}

			wg.Wait()

			select {
			case <-time.After(10 * time.Second):
				t.Fatalf("Timeout waiting for ReceiveContextCancelled status")
			case <-processingContextCanceled:
			}

			totalSum := 0
			for _, value := range m {
				totalSum += value
			}
			actualRatios := make([]float64, 4)
			for idx, value := range m {
				actualRatios[idx] = float64(value) / float64(totalSum)
			}
			expectedRatios := make([]float64, 4)
			var expectedRatiosSum int
			if tc.isSubset {
				expectedRatiosSum = 5 + 10
			} else {
				expectedRatiosSum = 1 + 3 + 5 + 10
			}
			freqRatios := []int{1, 3, 5, 10}
			for idx, value := range freqRatios {
				if tc.isSubset && (idx == 0 || idx == 1) {
					continue
				}
				expectedRatios[idx] = float64(value) / float64(expectedRatiosSum)
			}

			t.Logf("Total messages: %d", totalSum)
			for i := 0; i < 4; i++ {
				if tc.isSubset && (i == 0 || i == 1) {
					continue
				}
				diff := math.Abs(expectedRatios[i] - actualRatios[i])
				diffPercentage := (diff / expectedRatios[i]) * 100
				t.Logf("Channel %d: total messages %d, expected messages ratio %.5f, got %.5f (%.1f%%)",
					i, m[i], expectedRatios[i], actualRatios[i], diffPercentage)
			}
		})
	}
}

func TestProcessMessagesByFrequencyRatioWithGoroutines_StopsOnClosingAllChannels(t *testing.T) {
	channel1 := make(chan string)
	channel3 := make(chan string)
	channel5 := make(chan string)
	channel10 := make(chan string)

	channelsWithFreqRatios := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio("Channel 1", channel1, 1),
		channels.NewChannelWithFreqRatio("Channel 3", channel3, 3),
		channels.NewChannelWithFreqRatio("Channel 5", channel5, 5),
		channels.NewChannelWithFreqRatio("Channel 10", channel10, 10),
	}

	allChannels := []chan string{channel1, channel3, channel5, channel10}

	channelNameToIndex := map[string]int{
		"Channel 1":  0,
		"Channel 3":  1,
		"Channel 5":  2,
		"Channel 10": 3,
	}

	// senders
	for i := 0; i < 4; i++ {
		go func(c chan string, i int) {
			for j := 0; j < 1000; j++ {
				c <- fmt.Sprintf("message %d", j)
			}
			close(c)
		}(allChannels[i], i)
	}

	receiveCounts := make(map[int]int)
	closedChannels := make(map[int]struct{})
	var closedAllChannelsReceived atomic.Bool
	closedAllChannelsReceivedC := make(chan struct{})
	var mtx sync.Mutex
	onMessageReceived := func(msg string, channelName string) {
		mtx.Lock()
		idx := channelNameToIndex[channelName]
		receiveCounts[idx]++
		mtx.Unlock()
		// simulate some activity
		time.Sleep(50 * time.Microsecond)
	}
	onChannelClosed := func(channelName string) {
		mtx.Lock()
		if _, exists := closedChannels[channelNameToIndex[channelName]]; exists {
			t.Errorf("%s is already closed", channelName)
		}
		closedChannels[channelNameToIndex[channelName]] = struct{}{}
		mtx.Unlock()
	}
	onProcessingFinished := func(reason pc.ExitReason) {
		if reason == pc.NoOpenChannels {
			if closedAllChannelsReceived.Load() {
				t.Errorf("ReceiveNoOpenChannels status received multiple times")
			}
			closedAllChannelsReceivedC <- struct{}{}
		} else {
			t.Errorf("Unexpected processing finished reason: %v", reason)
		}
	}
	err := pc.ProcessByFrequencyRatioWithGoroutines(context.Background(), channelsWithFreqRatios,
		onMessageReceived, onChannelClosed, onProcessingFinished)
	if err != nil {
		t.Fatalf("Unexpected error on calling process by frequency ratio with goroutines: %v", err)
	}

	select {
	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for ReceiveNoOpenChannels status")
	case <-closedAllChannelsReceivedC:
		closedAllChannelsReceived.Store(true)
	}

	for i := 0; i < 4; i++ {
		if _, exists := closedChannels[i]; !exists {
			t.Errorf("Channel %d is not closed", i)
		}
		if receiveCounts[i] != 1000 {
			t.Errorf("Channel %d: expected 1000 messages, got %d", i, receiveCounts[i])
		}
	}
}

func TestProcessMessagesByFrequencyRatio_TenThousandMessages2(t *testing.T) {
	t.Skip()
	var testCases = []struct {
		Method          string
		FreqRatioMethod pc.FrequencyMethod
		MessagesRate    string
	}{
		{
			Method:          "Priority Channel - StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
			MessagesRate:    "Constant",
		},
		{
			Method:          "Priority Channel - StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
			MessagesRate:    "Constant",
		},
		{
			Method:          "Priority Channel - ProbabilisticByCaseDuplication",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
			MessagesRate:    "Constant",
		},
		{
			Method:          "Priority Channel - ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
			MessagesRate:    "Constant",
		},
		{
			Method:       "Select with Duplicate Cases",
			MessagesRate: "Constant",
		},
		{
			Method:          "Priority Channel - StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
			MessagesRate:    "Random",
		},
		{
			Method:          "Priority Channel - StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
			MessagesRate:    "Random",
		},
		{
			Method:          "Priority Channel - ProbabilisticByCaseDuplication",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
			MessagesRate:    "Random",
		},
		{
			Method:          "Priority Channel - ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
			MessagesRate:    "Random",
		},
		{
			Method:       "Select with Duplicate Cases",
			MessagesRate: "Random",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.Method+"-"+tc.MessagesRate), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var inputChannels []chan string
			var channelsWithFreqRatio []channels.ChannelWithFreqRatio[string]

			channelsNum := 5
			for i := 1; i <= channelsNum; i++ {
				//inputChannels = append(inputChannels, make(chan string, 10000))
				//inputChannels = append(inputChannels, make(chan string, 1000))
				inputChannels = append(inputChannels, make(chan string))
				channelsWithFreqRatio = append(channelsWithFreqRatio, channels.NewChannelWithFreqRatio(
					fmt.Sprintf("Channel %d", i), inputChannels[i-1], i))
			}

			freqTotalSum := 0.0
			for i := 1; i <= channelsNum; i++ {
				freqTotalSum += float64(channelsWithFreqRatio[i-1].FreqRatio())
			}
			expectedRatios := make(map[string]float64)
			for _, ch := range channelsWithFreqRatio {
				expectedRatios[ch.ChannelName()] = float64(ch.FreqRatio()) / freqTotalSum
			}

			messagesNum := 100000
			//messagesNum := 500

			if tc.MessagesRate == "Constant" {
				for i := 1; i <= channelsNum; i++ {
					go func(i int) {
						for j := 1; j <= messagesNum; j++ {
							select {
							case <-ctx.Done():
								return
							case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
							}
						}
					}(i)
				}
			} else if tc.MessagesRate == "Random" {
				go func() {
					for j := 1; j <= messagesNum; j++ {
						i := rand.IntN(channelsNum) + 1
						select {
						case <-ctx.Done():
							return
						case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
						}
					}
				}()
			}

			startTime := time.Now()

			totalCount := 0
			countPerChannel := make(map[string]int)

			if tc.Method == "Select with Duplicate Cases" {
				//go func() {
				//	for {
				//		var channel string
				//		select {
				//		case <-ctx.Done():
				//			return
				//		case <-inputChannels[0]:
				//			channel = "Channel A"
				//		case <-inputChannels[1]:
				//			channel = "Channel B"
				//		case <-inputChannels[1]:
				//			channel = "Channel B"
				//		case <-inputChannels[2]:
				//			channel = "Channel C"
				//		case <-inputChannels[2]:
				//			channel = "Channel C"
				//		case <-inputChannels[2]:
				//			channel = "Channel C"
				//		case <-inputChannels[3]:
				//			channel = "Channel D"
				//		case <-inputChannels[3]:
				//			channel = "Channel D"
				//		case <-inputChannels[3]:
				//			channel = "Channel D"
				//		case <-inputChannels[3]:
				//			channel = "Channel D"
				//		}
				//		totalCount++
				//		countPerChannel[channel] = countPerChannel[channel] + 1
				//		if totalCount == messagesNum {
				//			cancel()
				//			return
				//		}
				//		time.Sleep(100 * time.Microsecond)
				//	}
				//}()
				selectCases := make([]reflect.SelectCase, 0, channelsNum+1)
				selectCases = append(selectCases, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(ctx.Done()),
				})
				for i := 1; i <= channelsNum; i++ {
					for j := 1; j <= i; j++ {
						selectCases = append(selectCases, reflect.SelectCase{
							Dir:  reflect.SelectRecv,
							Chan: reflect.ValueOf(inputChannels[i-1]),
						})
					}
				}

				go func() {
					for {
						//rand.Shuffle(len(selectCases), func(i int, j int) {
						//	selectCases[i], selectCases[j] = selectCases[j], selectCases[i]
						//})
						_, recv, recvOk := reflect.Select(selectCases)
						if !recvOk {
							return
						}
						channel, _ := recv.Interface().(string)
						totalCount++
						countPerChannel[channel] = countPerChannel[channel] + 1
						if totalCount == messagesNum {
							cancel()
							return
						}
						time.Sleep(1 * time.Microsecond)
					}
				}()

			} else if strings.HasPrefix(tc.Method, "Priority Channel") {
				ch, _ := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio, pc.WithFrequencyMethod(tc.FreqRatioMethod))
				go func() {
					for {
						_, channel, ok := ch.Receive()
						if !ok {
							return
						}
						time.Sleep(1 * time.Microsecond)
						totalCount++
						countPerChannel[channel] = countPerChannel[channel] + 1
						if totalCount == messagesNum {
							cancel()
							return
						}
					}
				}()
			}

			<-ctx.Done()

			elapsedTime := time.Since(startTime)
			t.Logf("Elapsed time: %v\n", elapsedTime)
			//  1.832873667s    for 1M messages - freq ratio priority channel with 100 microseconds wait
			//  1m57.941205209s for 1M messages - duplicated  channel with 100 microseconds wait
			//  2.045487917s for 1M messages - freq ratio priority channel with 5 microseconds wait
			//  4.171439583s for 1M messages - duplicated  channel with 1 microseconds wait

			for _, channel := range channelsWithFreqRatio {
				expectedRatio := expectedRatios[channel.ChannelName()]
				actualRatio := float64(countPerChannel[channel.ChannelName()]) / float64(totalCount)
				if math.Abs(expectedRatio-actualRatio) > 0.01 {
					t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
						channel.ChannelName(), expectedRatio, actualRatio)
				}
			}
		})
	}
}

func TestProcessMessagesByFrequencyRatio_TenThousandMessages3(t *testing.T) {
	t.Skip()
	ctx, cancel := context.WithCancel(context.Background())
	var inputChannels []chan string

	channelsNum := 4
	for i := 1; i <= channelsNum; i++ {
		inputChannels = append(inputChannels, make(chan string))
	}

	channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio("Channel A", inputChannels[0], 1),
		channels.NewChannelWithFreqRatio("Channel B", inputChannels[1], 2),
		channels.NewChannelWithFreqRatio("Channel C", inputChannels[2], 3),
		channels.NewChannelWithFreqRatio("Channel D", inputChannels[3], 4),
		//channels.NewChannelWithFreqRatio("Channel E", inputChannels[4], 5),
	}

	freqTotalSum := 0.0
	for i := 1; i <= channelsNum; i++ {
		freqTotalSum += float64(channelsWithFreqRatio[i-1].FreqRatio())
	}
	expectedRatios := make(map[string]float64)
	for _, ch := range channelsWithFreqRatio {
		expectedRatios[ch.ChannelName()] = float64(ch.FreqRatio()) / freqTotalSum
	}

	for i := 1; i <= channelsNum; i++ {
		go func(i int) {
			for j := 1; j <= 10000; j++ {
				select {
				case <-ctx.Done():
					return
				case inputChannels[i-1] <- channelsWithFreqRatio[i-1].ChannelName():
				}
			}
		}(i)
	}

	totalCount := 0
	countPerChannel := make(map[string]int)

	selectCases := make([]reflect.SelectCase, 0, channelsNum+1)
	selectCases = append(selectCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})
	for i := 1; i <= channelsNum; i++ {
		for j := 1; j <= i; j++ {
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(inputChannels[i-1]),
			})
		}
	}

	go func() {
		for {
			rand.Shuffle(len(selectCases), func(i int, j int) {
				selectCases[i], selectCases[j] = selectCases[j], selectCases[i]
			})
			_, recv, recvOk := reflect.Select(selectCases)
			if !recvOk {
				return
			}
			channel, _ := recv.Interface().(string)
			totalCount++
			countPerChannel[channel] = countPerChannel[channel] + 1
			if totalCount == 10000 {
				cancel()
				return
			}
		}
	}()

	//ch, _ := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio)
	//go func() {
	//	for {
	//		_, channel, ok := ch.Receive()
	//		if !ok {
	//			return
	//		}
	//		totalCount++
	//		countPerChannel[channel] = countPerChannel[channel] + 1
	//		if totalCount == 10000 {
	//			cancel()
	//			return
	//		}
	//	}
	//}()

	<-ctx.Done()

	for _, channel := range channelsWithFreqRatio {
		expectedRatio := expectedRatios[channel.ChannelName()]
		actualRatio := float64(countPerChannel[channel.ChannelName()]) / float64(totalCount)
		if math.Abs(expectedRatio-actualRatio) > 0.03 {
			t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
				channel.ChannelName(), expectedRatio, actualRatio)
		}
	}
}

func TestProcessMessagesByFrequencyRatio_TenThousandMessages4(t *testing.T) {
	t.Skip()
	var testCases = []struct {
		Method          string
		FreqRatioMethod pc.FrequencyMethod
		MessagesRate    string
	}{
		{
			Method:          "Priority Channel - StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
			MessagesRate:    "Constant",
		},
		{
			Method:          "Priority Channel - StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
			MessagesRate:    "Constant",
		},
		{
			Method:          "Priority Channel - ProbabilisticByCaseDuplication",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
			MessagesRate:    "Constant",
		},
		{
			Method:          "Priority Channel - ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
			MessagesRate:    "Constant",
		},
		{
			Method:       "Select with Duplicate Cases",
			MessagesRate: "Constant",
		},
		{
			Method:          "Priority Channel - StrictOrderAcrossCycles",
			FreqRatioMethod: pc.StrictOrderAcrossCycles,
			MessagesRate:    "Random",
		},
		{
			Method:          "Priority Channel - StrictOrderFully",
			FreqRatioMethod: pc.StrictOrderFully,
			MessagesRate:    "Random",
		},
		{
			Method:          "Priority Channel - ProbabilisticByCaseDuplication",
			FreqRatioMethod: pc.ProbabilisticByCaseDuplication,
			MessagesRate:    "Random",
		},
		{
			Method:          "Priority Channel - ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: pc.ProbabilisticByMultipleRandCalls,
			MessagesRate:    "Random",
		},
		{
			Method:       "Select with Duplicate Cases",
			MessagesRate: "Random",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.Method+"-"+tc.MessagesRate), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var inputChannels []chan string
			channelsNum := 2
			for i := 1; i <= channelsNum; i++ {
				inputChannels = append(inputChannels, make(chan string))
			}
			channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio("Channel 1", inputChannels[0], 37),
				channels.NewChannelWithFreqRatio("Channel 2", inputChannels[1], 63),
			}

			freqTotalSum := 0.0
			for i := 1; i <= channelsNum; i++ {
				freqTotalSum += float64(channelsWithFreqRatio[i-1].FreqRatio())
			}
			expectedRatios := make(map[string]float64)
			for _, ch := range channelsWithFreqRatio {
				expectedRatios[ch.ChannelName()] = float64(ch.FreqRatio()) / freqTotalSum
			}

			messagesNum := 10000
			//messagesNum := 500

			if tc.MessagesRate == "Constant" {
				for i := 1; i <= channelsNum; i++ {
					go func(i int) {
						for j := 1; j <= messagesNum; j++ {
							select {
							case <-ctx.Done():
								return
							case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
							}
						}
					}(i)
				}
			} else if tc.MessagesRate == "Random" {
				go func() {
					for j := 1; j <= messagesNum; j++ {
						i := rand.IntN(channelsNum) + 1
						select {
						case <-ctx.Done():
							return
						case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
						}
					}
				}()
			}

			startTime := time.Now()

			totalCount := 0
			countPerChannel := make(map[string]int)

			if tc.Method == "Select with Duplicate Cases" {
				selectCases := make([]reflect.SelectCase, 0, channelsNum+1)
				selectCases = append(selectCases, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(ctx.Done()),
				})
				for i := 1; i <= channelsNum; i++ {
					for j := 1; j <= i; j++ {
						selectCases = append(selectCases, reflect.SelectCase{
							Dir:  reflect.SelectRecv,
							Chan: reflect.ValueOf(inputChannels[i-1]),
						})
					}
				}
				go func() {
					for {
						_, recv, recvOk := reflect.Select(selectCases)
						if !recvOk {
							return
						}
						channel, _ := recv.Interface().(string)
						totalCount++
						countPerChannel[channel] = countPerChannel[channel] + 1
						if totalCount == messagesNum {
							cancel()
							return
						}
						time.Sleep(5 * time.Microsecond)
					}
				}()

			} else if strings.HasPrefix(tc.Method, "Priority Channel") {
				ch, _ := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio, pc.WithFrequencyMethod(tc.FreqRatioMethod))
				go func() {
					for {
						_, channel, ok := ch.Receive()
						if !ok {
							return
						}
						time.Sleep(5 * time.Microsecond)
						totalCount++
						countPerChannel[channel] = countPerChannel[channel] + 1
						if totalCount == messagesNum {
							cancel()
							return
						}
					}
				}()
			}

			<-ctx.Done()

			elapsedTime := time.Since(startTime)
			t.Logf("Elapsed time: %v\n", elapsedTime)

			for _, channel := range channelsWithFreqRatio {
				expectedRatio := expectedRatios[channel.ChannelName()]
				actualRatio := float64(countPerChannel[channel.ChannelName()]) / float64(totalCount)
				if math.Abs(expectedRatio-actualRatio) > 0.01 {
					t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
						channel.ChannelName(), expectedRatio, actualRatio)
				}
			}
		})
	}
}

func TestProcessMessagesByFrequencyRatio_HighPriorityAlwaysFirstOverDistinctNumberOfPriorities(t *testing.T) {
	t.Skip()
	var testCaseChannelsNum = []int{5, 10, 50, 100, 250, 500, 1000}

	for _, tc := range testCaseChannelsNum {
		t.Run(fmt.Sprintf("Channels num - %d", tc), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			var inputChannels []chan string
			var channelsWithPriority []channels.ChannelWithPriority[string]

			channelsNum := tc
			for i := 1; i <= channelsNum; i++ {
				//inputChannels = append(inputChannels, make(chan string, 10000))
				//inputChannels = append(inputChannels, make(chan string, 1000))
				inputChannels = append(inputChannels, make(chan string))
				channelsWithPriority = append(channelsWithPriority, channels.NewChannelWithPriority(
					fmt.Sprintf("Channel %d", i), inputChannels[i-1], i))
			}

			messagesNum := 1000
			//messagesNum := 100000
			//messagesNum := 500

			go func() {
				for j := 1; j <= messagesNum; j++ {
					select {
					case <-ctx.Done():
						return
					case inputChannels[0] <- fmt.Sprintf("New message"):
					}
				}
			}()

			startTime := time.Now()

			totalCount := 0
			countPerChannel := make(map[string]int)

			ch, _ := pc.NewByHighestAlwaysFirst(ctx, channelsWithPriority)
			go func() {
				for {
					_, channel, ok := ch.Receive()
					if !ok {
						return
					}
					time.Sleep(5 * time.Microsecond)
					totalCount++
					countPerChannel[channel] = countPerChannel[channel] + 1
					if totalCount == messagesNum {
						cancel()
						return
					}
				}
			}()

			<-ctx.Done()

			elapsedTime := time.Since(startTime)
			t.Logf("Elapsed time: %v\n", elapsedTime)
		})
	}
}

func TestProcessMessagesByFrequencyRatio_SingleSelectCaseOverLargeNumberOfChannels(t *testing.T) {
	t.Skip()
	var testCaseChannelsNum = []int{5, 10, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 20000, 25000, 65000, 70000}

	for _, tc := range testCaseChannelsNum {
		t.Run(fmt.Sprintf("Channels num - %d", tc), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			var inputChannels []chan string
			var channelsWithPriority []channels.ChannelWithPriority[string]

			channelsNum := tc
			for i := 1; i <= channelsNum; i++ {
				//inputChannels = append(inputChannels, make(chan string, 10000))
				//inputChannels = append(inputChannels, make(chan string, 1000))
				inputChannels = append(inputChannels, make(chan string))
				channelsWithPriority = append(channelsWithPriority, channels.NewChannelWithPriority(
					fmt.Sprintf("Channel %d", i), inputChannels[i-1], i))
			}

			messagesNum := 1000
			//messagesNum := 500

			go func() {
				for j := 1; j <= messagesNum; j++ {
					i := rand.IntN(channelsNum) + 1
					select {
					case <-ctx.Done():
						return
					case inputChannels[i-1] <- fmt.Sprintf("Channel %d", i):
					}
				}
			}()

			startTime := time.Now()

			totalCount := 0
			countPerChannel := make(map[string]int)

			selectCases := make([]reflect.SelectCase, 0, channelsNum+1)
			selectCases = append(selectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ctx.Done()),
			})
			for i := 1; i <= channelsNum; i++ {
				selectCases = append(selectCases, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(inputChannels[i-1]),
				})
			}

			go func() {
				for {
					_, recv, recvOk := reflect.Select(selectCases)
					if !recvOk {
						return
					}
					channel, _ := recv.Interface().(string)
					totalCount++
					countPerChannel[channel] = countPerChannel[channel] + 1
					if totalCount == messagesNum {
						cancel()
						return
					}
					time.Sleep(5 * time.Microsecond)
				}
			}()

			<-ctx.Done()

			elapsedTime := time.Since(startTime)
			t.Logf("Elapsed time: %v\n", elapsedTime)
		})
	}
}

func TestProcessMessagesByFrequencyRatio_AutoDisableClosedChannels(t *testing.T) {
	testCases := []struct {
		name            string
		frequencyMethod pc.FrequencyMethod
	}{
		{
			name:            "StrictOrderAcrossCycles",
			frequencyMethod: pc.StrictOrderAcrossCycles,
		},
		{
			name:            "StrictOrderFully",
			frequencyMethod: pc.StrictOrderFully,
		},
		{
			name:            "ProbabilisticByCaseDuplication",
			frequencyMethod: pc.ProbabilisticByCaseDuplication,
		},
		{
			name:            "ProbabilisticByMultipleRandCalls",
			frequencyMethod: pc.ProbabilisticByMultipleRandCalls,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			urgentMessagesC := make(chan string)
			highPriorityC := make(chan string)
			normalPriorityC := make(chan string)
			lowPriorityC := make(chan string)

			// sending messages to individual channels
			go func() {
				for i := 1; i <= 50; i++ {
					highPriorityC <- fmt.Sprintf("high priority message %d", i)
				}
				close(highPriorityC)
			}()
			go func() {
				for i := 1; i <= 50; i++ {
					normalPriorityC <- fmt.Sprintf("normal priority message %d", i)
				}
				close(normalPriorityC)
			}()
			go func() {
				for i := 1; i <= 50; i++ {
					lowPriorityC <- fmt.Sprintf("low priority message %d", i)
				}
				close(lowPriorityC)
			}()
			go func() {
				for i := 1; i <= 50; i++ {
					urgentMessagesC <- fmt.Sprintf("urgent message %d", i)
				}
				close(urgentMessagesC)
			}()

			channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"High Priority",
					highPriorityC,
					8),
				channels.NewChannelWithFreqRatio(
					"Normal Priority",
					normalPriorityC,
					5),
				channels.NewChannelWithFreqRatio(
					"Low Priority",
					lowPriorityC,
					3),
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					urgentMessagesC,
					10),
			}
			ch, err := pc.NewByFrequencyRatio(ctx, channelsWithFreqRatio,
				pc.AutoDisableClosedChannels(),
				pc.WithFrequencyMethod(tc.frequencyMethod))
			if err != nil {
				t.Fatalf("Unexpected error on priority channel intialization: %v", err)
			}

			receivedMessagesCount := 0
			for {
				message, channelName, status := ch.ReceiveWithContext(context.Background())
				if status != pc.ReceiveSuccess {
					if receivedMessagesCount != 200 {
						t.Errorf("Expected to receive 200 messages, but got %d", receivedMessagesCount)
					}
					if status != pc.ReceiveNoOpenChannels {
						t.Errorf("Expected to receive 'no open channels' status on closure (%v), but got %v",
							pc.ReceiveNoOpenChannels, status)
					}
					break
				}
				receivedMessagesCount++
				fmt.Printf("%s: %s\n", channelName, message)
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestProcessMessagesByFrequencyRatio_MessagesInOneOfTheChannelsArriveAfterSomeTime(t *testing.T) {
	msgsChannels := make([]chan *Msg, 3)
	msgsChannels[0] = make(chan *Msg, 7)
	msgsChannels[1] = make(chan *Msg, 7)
	msgsChannels[2] = make(chan *Msg, 7)

	channels := []channels.ChannelWithFreqRatio[*Msg]{
		channels.NewChannelWithFreqRatio("Priority-1", msgsChannels[0], 1),
		channels.NewChannelWithFreqRatio("Priority-2", msgsChannels[1], 2),
		channels.NewChannelWithFreqRatio("Priority-3", msgsChannels[2], 3),
	}

	simulateLongProcessingMsg := "Simulate long processing"
	for j := 1; j <= 5; j++ {
		msgsChannels[0] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[0].ChannelName(), j)}
		suffix := ""
		if j == 5 {
			suffix = " - " + simulateLongProcessingMsg
		}
		msgsChannels[2] <- &Msg{Body: fmt.Sprintf("%s Msg-%d%s", channels[2].ChannelName(), j, suffix)}
	}

	waitForMessagesFromPriority2Chan := make(chan struct{})
	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		if strings.HasSuffix(msg.Body, simulateLongProcessingMsg) {
			<-waitForMessagesFromPriority2Chan
		}
		results = append(results, msg)
	}
	ctx, cancel := context.WithCancel(context.Background())

	priorityChannel, err := pc.NewByFrequencyRatio(ctx, channels, pc.WithFrequencyMethod(pc.StrictOrderFully))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	go pc.ProcessPriorityChannelMessages(priorityChannel, msgProcessor)

	time.Sleep(1 * time.Second)
	for j := 6; j <= 7; j++ {
		msgsChannels[0] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[0].ChannelName(), j)}
		msgsChannels[2] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[2].ChannelName(), j)}
	}
	msgsChannels[1] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[1].ChannelName(), 1)}
	msgsChannels[1] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[1].ChannelName(), 2)}
	msgsChannels[1] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[1].ChannelName(), 3)}
	waitForMessagesFromPriority2Chan <- struct{}{}

	time.Sleep(3 * time.Second)
	cancel()

	expectedResults := []*Msg{
		{Body: "Priority-3 Msg-1"},
		{Body: "Priority-3 Msg-2"},
		{Body: "Priority-3 Msg-3"},
		{Body: "Priority-1 Msg-1"},
		{Body: "Priority-3 Msg-4"},
		{Body: "Priority-3 Msg-5 - Simulate long processing"},
		{Body: "Priority-2 Msg-1"},
		{Body: "Priority-2 Msg-2"},
		{Body: "Priority-3 Msg-6"},
		{Body: "Priority-2 Msg-3"},
		{Body: "Priority-1 Msg-2"},
		{Body: "Priority-3 Msg-7"},
		{Body: "Priority-1 Msg-3"},
		{Body: "Priority-1 Msg-4"},
		{Body: "Priority-1 Msg-5"},
		{Body: "Priority-1 Msg-6"},
		{Body: "Priority-1 Msg-7"},
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, but got %d", len(expectedResults), len(results))
	}
	for i := range results {
		if results[i].Body != expectedResults[i].Body {
			t.Errorf("Result %d: Expected message %s, but got %s",
				i, expectedResults[i].Body, results[i].Body)
		}
	}
}

func TestProcessMessagesByFrequencyRatio_ChannelClosed(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	close(normalPriorityC)

	ch, err := pc.NewByFrequencyRatio(context.Background(), channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	for i := 0; i < 3; i++ {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != pc.ReceiveChannelClosed {
			t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", pc.ReceiveChannelClosed, status)
		}
		if channelName != "Normal Priority" {
			t.Errorf("Expected channel name 'Normal Priority', but got %s", channelName)
		}
		if message != "" {
			t.Errorf("Expected empty message, but got %s", message)
		}
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != pc.ReceiveChannelClosed {
		t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", pc.ReceiveChannelClosed, status)
	}
	if channelName != "Normal Priority" {
		t.Errorf("Expected channel name 'Normal Priority', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByFrequencyRatio_ExitOnDefaultCase(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	ch, err := pc.NewByFrequencyRatio(context.Background(), channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != pc.ReceiveDefaultCase {
		t.Errorf("Expected status ReceiveDefaultCase (%d), but got %d", pc.ReceiveDefaultCase, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByFrequencyRatio_RequestContextCancelled(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	ch, err := pc.NewByFrequencyRatio(context.Background(), channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	message, channelName, status := ch.ReceiveWithContext(ctx)
	if status != pc.ReceiveContextCancelled {
		t.Errorf("Expected status ReceiveContextCancelled (%d), but got %d", pc.ReceiveContextCancelled, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestProcessMessagesByFrequencyRatio_PriorityChannelContextCancelled(t *testing.T) {
	highPriorityC := make(chan string)
	normalPriorityC := make(chan string)
	lowPriorityC := make(chan string)

	channelsWithFrequencyRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"High Priority",
			highPriorityC,
			10),
		channels.NewChannelWithFreqRatio(
			"Normal Priority",
			normalPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Low Priority",
			lowPriorityC,
			1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := pc.NewByFrequencyRatio(ctx, channelsWithFrequencyRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	message, channelName, status := ch.ReceiveWithContext(context.Background())
	if status != pc.ReceivePriorityChannelClosed {
		t.Errorf("Expected status ReceivePriorityChannelClosed (%d), but got %d", pc.ReceivePriorityChannelClosed, status)
	}
	if channelName != "" {
		t.Errorf("Expected empty channel name, but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestByFrequencyRatioPriorityChannelValidation(t *testing.T) {
	var testCases = []struct {
		Name                   string
		ChannelsWithFreqRatios []channels.ChannelWithFreqRatio[string]
		ExpectedErrorMessage   string
	}{
		{
			Name:                   "No channels",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{},
			ExpectedErrorMessage:   pc.ErrNoChannels.Error(),
		},
		{
			Name: "Empty channel name",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					5),
				channels.NewChannelWithFreqRatio(
					"",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: pc.ErrEmptyChannelName.Error(),
		},
		{
			Name: "Zero frequency ratio value",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					0),
				channels.NewChannelWithFreqRatio(
					"Low Priority Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel 'Normal Messages': frequency ratio must be greater than 0",
		},
		{
			Name: "Negative frequency ratio value",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					-5),
				channels.NewChannelWithFreqRatio(
					"Low Priority Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel 'Normal Messages': frequency ratio must be greater than 0",
		},
		{
			Name: "Duplicate channel name",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					10),
				channels.NewChannelWithFreqRatio(
					"Normal Messages",
					make(chan string),
					5),
				channels.NewChannelWithFreqRatio(
					"Urgent Messages",
					make(chan string),
					1),
			},
			ExpectedErrorMessage: "channel name 'Urgent Messages' is used more than once",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			_, err := pc.NewByFrequencyRatio(ctx, tc.ChannelsWithFreqRatios)
			if tc.ExpectedErrorMessage == "" {
				if err != nil {
					t.Fatalf("Unexpected validation error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Expected validation error")
			}
			if err.Error() != tc.ExpectedErrorMessage {
				t.Errorf("Expected error %v, but got: %v", tc.ExpectedErrorMessage, err)
			}
		})
	}
}
