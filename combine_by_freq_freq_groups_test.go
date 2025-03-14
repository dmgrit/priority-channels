package priority_channels_test

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func TestProcessMessagesByFreqRatioAmongFreqRatioChannelGroups(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	options := priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully)
	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	}, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	}, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio("Free User",
			freeUserPriorityChannel,
			1),
	}

	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	// sending messages to individual channels
	go func() {
		for i := 1; i <= 20; i++ {
			payingCustomerHighPriorityC <- fmt.Sprintf("high priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			payingCustomerLowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			freeUserHighPriorityC <- fmt.Sprintf("high priority message %d", i)
		}
	}()
	go func() {
		for i := 1; i <= 20; i++ {
			freeUserLowPriorityC <- fmt.Sprintf("low priority message %d", i)
		}
	}()

	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	// receiving messages from the priority channel
	results := make([]string, 0, 80)
	for {
		message, channelName, ok := ch.Receive()
		if !ok {
			break
		}
		fmt.Printf("%s: %s\n", channelName, message)
		results = append(results, fmt.Sprintf("%s: %s", channelName, message))
		time.Sleep(5 * time.Millisecond)
	}

	expectedResults := []string{
		"Paying Customer - High Priority: high priority message 1",
		"Paying Customer - High Priority: high priority message 2",
		"Paying Customer - High Priority: high priority message 3",
		"Paying Customer - High Priority: high priority message 4",
		"Paying Customer - High Priority: high priority message 5",
		"Paying Customer - Low Priority: low priority message 1",
		"Paying Customer - High Priority: high priority message 6",
		"Paying Customer - High Priority: high priority message 7",
		"Paying Customer - High Priority: high priority message 8",
		"Paying Customer - High Priority: high priority message 9",
		"Free User - High Priority: high priority message 1",
		"Paying Customer - High Priority: high priority message 10",
		"Paying Customer - Low Priority: low priority message 2",
		"Paying Customer - High Priority: high priority message 11",
		"Paying Customer - High Priority: high priority message 12",
		"Paying Customer - High Priority: high priority message 13",
		"Paying Customer - High Priority: high priority message 14",
		"Paying Customer - High Priority: high priority message 15",
		"Paying Customer - Low Priority: low priority message 3",
		"Paying Customer - High Priority: high priority message 16",
		"Paying Customer - High Priority: high priority message 17",
		"Free User - High Priority: high priority message 2",
		"Paying Customer - High Priority: high priority message 18",
		"Paying Customer - High Priority: high priority message 19",
		"Paying Customer - High Priority: high priority message 20",
		"Paying Customer - Low Priority: low priority message 4",
		"Paying Customer - Low Priority: low priority message 5",
		"Paying Customer - Low Priority: low priority message 6",
		"Paying Customer - Low Priority: low priority message 7",
		"Paying Customer - Low Priority: low priority message 8",
		"Paying Customer - Low Priority: low priority message 9",
		"Paying Customer - Low Priority: low priority message 10",
		"Free User - High Priority: high priority message 3",
		"Paying Customer - Low Priority: low priority message 11",
		"Paying Customer - Low Priority: low priority message 12",
		"Paying Customer - Low Priority: low priority message 13",
		"Paying Customer - Low Priority: low priority message 14",
		"Paying Customer - Low Priority: low priority message 15",
		"Paying Customer - Low Priority: low priority message 16",
		"Paying Customer - Low Priority: low priority message 17",
		"Paying Customer - Low Priority: low priority message 18",
		"Paying Customer - Low Priority: low priority message 19",
		"Paying Customer - Low Priority: low priority message 20",
		"Free User - High Priority: high priority message 4",
		"Free User - High Priority: high priority message 5",
		"Free User - Low Priority: low priority message 1",
		"Free User - High Priority: high priority message 6",
		"Free User - High Priority: high priority message 7",
		"Free User - High Priority: high priority message 8",
		"Free User - High Priority: high priority message 9",
		"Free User - High Priority: high priority message 10",
		"Free User - Low Priority: low priority message 2",
		"Free User - High Priority: high priority message 11",
		"Free User - High Priority: high priority message 12",
		"Free User - High Priority: high priority message 13",
		"Free User - High Priority: high priority message 14",
		"Free User - High Priority: high priority message 15",
		"Free User - Low Priority: low priority message 3",
		"Free User - High Priority: high priority message 16",
		"Free User - High Priority: high priority message 17",
		"Free User - High Priority: high priority message 18",
		"Free User - High Priority: high priority message 19",
		"Free User - High Priority: high priority message 20",
		"Free User - Low Priority: low priority message 4",
		"Free User - Low Priority: low priority message 5",
		"Free User - Low Priority: low priority message 6",
		"Free User - Low Priority: low priority message 7",
		"Free User - Low Priority: low priority message 8",
		"Free User - Low Priority: low priority message 9",
		"Free User - Low Priority: low priority message 10",
		"Free User - Low Priority: low priority message 11",
		"Free User - Low Priority: low priority message 12",
		"Free User - Low Priority: low priority message 13",
		"Free User - Low Priority: low priority message 14",
		"Free User - Low Priority: low priority message 15",
		"Free User - Low Priority: low priority message 16",
		"Free User - Low Priority: low priority message 17",
		"Free User - Low Priority: low priority message 18",
		"Free User - Low Priority: low priority message 19",
		"Free User - Low Priority: low priority message 20",
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, but got %d", len(expectedResults), len(results))
	}
	for i := range results {
		if results[i] != expectedResults[i] {
			t.Errorf("Result %d: Expected message %s, but got %s",
				i, expectedResults[i], results[i])
		}
	}
}

func TestCombinePriorityChannelsByFreqRatio_ErrorOnInvalidFrequencyMethod(t *testing.T) {
	ctx := context.Background()
	customerAPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx,
		"Customer A", make(chan string))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	customerBPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx,
		"Customer B", make(chan string))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	_, err = priority_channels.CombineByFrequencyRatio[string](ctx, []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio("Customer A", customerAPriorityChannel, 5),
		priority_channels.NewPriorityChannelWithFreqRatio("Customer B", customerBPriorityChannel, 1),
	}, priority_channels.WithFrequencyMethod(22))

	if err == nil {
		t.Fatalf("Expected invalid frequency method error but got none")
	}
	if err.Error() != priority_channels.ErrInvalidFrequencyMethod.Error() {
		t.Fatalf("Expected invalid frequency method but got %v", err)
	}
}

func TestProcessMessagesByFreqRatioAmongFreqRatioChannelGroups_TenThousandMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	payingCustomerHighPriorityFlagshipProductC := make(chan string)
	payingCustomerHighPriorityNicheProductC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	inputChannels := []chan string{
		payingCustomerHighPriorityFlagshipProductC,
		payingCustomerHighPriorityNicheProductC,
		payingCustomerLowPriorityC,
		freeUserHighPriorityC,
		freeUserLowPriorityC,
	}

	payingCustomerHighPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority - Flagship Product",
			payingCustomerHighPriorityFlagshipProductC,
			3),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority - Niche Product",
			payingCustomerHighPriorityNicheProductC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	payingCustomerLowPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx,
		"Paying Customer - Low Priority", payingCustomerLowPriorityC)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	payingCustomerPriorityChannel, err := priority_channels.CombineByFrequencyRatio[string](ctx, []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityChannel,
			4),
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityChannel,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			4),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithPriority := []priority_channels.PriorityChannelWithPriority[string]{
		priority_channels.NewPriorityChannelWithPriority("Paying Customer",
			payingCustomerPriorityChannel,
			5),
		priority_channels.NewPriorityChannelWithPriority("Free User",
			freeUserPriorityChannel,
			5),
	}
	//channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
	//	priority_channels.NewPriorityChannelWithFreqRatio("Paying Customer",
	//		payingCustomerPriorityChannel,
	//		5),
	//	priority_channels.NewPriorityChannelWithFreqRatio("Free User",
	//		freeUserPriorityChannel,
	//		5),
	//}

	expectedRatios := map[string]float64{
		"Paying Customer - High Priority - Flagship Product": 0.3,
		"Paying Customer - High Priority - Niche Product":    0.1,
		"Paying Customer - Low Priority":                     0.1,
		"Free User - High Priority":                          0.4,
		"Free User - Low Priority":                           0.1,
	}
	messagesNum := 10000

	for i := range inputChannels {
		go func(i int) {
			for j := 1; j <= messagesNum; j++ {
				select {
				case <-ctx.Done():
					return
				case inputChannels[i] <- fmt.Sprintf("Message %d", i):
				}
			}
		}(i)
	}

	ch, err := priority_channels.CombineByHighestAlwaysFirst[string](ctx, channelsWithPriority)
	//options := priority_channels.WithFrequencyMethod(priority_channels.ProbabilisticByCaseDuplication)
	//ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio, options)
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
			totalCount++
			time.Sleep(10 * time.Microsecond)
			countPerChannel[channel] = countPerChannel[channel] + 1
			if totalCount == messagesNum {
				cancel()
				return
			}
		}
	}()

	<-ctx.Done()

	for channelName := range expectedRatios {
		expectedRatio := expectedRatios[channelName]
		actualRatio := float64(countPerChannel[channelName]) / float64(totalCount)
		if math.Abs(expectedRatio-actualRatio) > 0.03 {
			t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
				channelName, expectedRatio, actualRatio)
		}
	}
}

func TestProcessMessagesByFreqRatioAmongFreqRatioChannelGroups_TenThousandMessages_Sanity(t *testing.T) {
	var testCases = []struct {
		Name            string
		FreqRatioMethod priority_channels.FrequencyMethod
	}{
		{
			Name:            "StrictOrderAcrossCycles",
			FreqRatioMethod: priority_channels.StrictOrderAcrossCycles,
		},
		{
			Name:            "StrictOrderFully",
			FreqRatioMethod: priority_channels.StrictOrderFully,
		},
		{
			Name:            "ProbabilisticWithCasesDuplications",
			FreqRatioMethod: priority_channels.ProbabilisticByCaseDuplication,
		},
		{
			Name:            "ProbabilisticByMultipleRandCalls",
			FreqRatioMethod: priority_channels.ProbabilisticByMultipleRandCalls,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testProcessMessagesOfCombinedPriorityChannelsByFrequencyRatioWithMethod(t, tc.FreqRatioMethod, 10000)
		})
	}
}

func testProcessMessagesOfCombinedPriorityChannelsByFrequencyRatioWithMethod(t *testing.T, freqRatioMethod priority_channels.FrequencyMethod, messagesNum int) {
	ctx, cancel := context.WithCancel(context.Background())
	var inputChannels []chan string
	var priorityChannelsWithFreqRatio []priority_channels.PriorityChannelWithFreqRatio[string]

	channelsNum := 5
	for i := 1; i <= channelsNum; i++ {
		inputChannel := make(chan string)
		inputChannels = append(inputChannels, inputChannel)
		ch, err := priority_channels.WrapAsPriorityChannel(ctx, fmt.Sprintf("channel-%d", i), inputChannel)
		if err != nil {
			t.Fatalf("Unexpected error on priority channel intialization: %v", err)
		}
		priorityChannelsWithFreqRatio = append(priorityChannelsWithFreqRatio, priority_channels.NewPriorityChannelWithFreqRatio(
			fmt.Sprintf("priority-channel-%d", i), ch, i))
	}

	freqTotalSum := 0.0
	for i := 1; i <= channelsNum; i++ {
		freqTotalSum += float64(priorityChannelsWithFreqRatio[i-1].FreqRatio())
	}
	expectedRatios := make(map[string]float64)
	for _, ch := range priorityChannelsWithFreqRatio {
		channelName := strings.TrimPrefix(ch.Name(), "priority-")
		expectedRatios[channelName] = float64(ch.FreqRatio()) / freqTotalSum
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

	ch, err := priority_channels.CombineByFrequencyRatio(ctx, priorityChannelsWithFreqRatio, priority_channels.WithFrequencyMethod(freqRatioMethod))
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

	for _, channel := range priorityChannelsWithFreqRatio {
		channelName := strings.TrimPrefix(channel.Name(), "priority-")
		expectedRatio := expectedRatios[channelName]
		actualRatio := float64(countPerChannel[channelName]) / float64(totalCount)
		if math.Abs(expectedRatio-actualRatio) > 0.03 {
			t.Errorf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
				channelName, expectedRatio, actualRatio)
		}
		t.Logf("Channel %s: expected messages number by ratio %.2f, got %.2f\n",
			channelName, expectedRatio, actualRatio)
	}
}

type channelWithExpectedRatio struct {
	channel       chan string
	expectedRatio float64
}

func TestProcessMessagesOfCombinedPriorityChannelsByFrequencyRatio_RandomTree(t *testing.T) {
	t.Skip()
	var testCases = []struct {
		Name            string
		FrequencyMethod priority_channels.FrequencyMethod
	}{
		{
			Name:            "StrictOrderAcrossCycles",
			FrequencyMethod: priority_channels.StrictOrderAcrossCycles,
		},
		{
			Name:            "StrictOrderFully",
			FrequencyMethod: priority_channels.StrictOrderFully,
		},
		{
			Name:            "ProbabilisticWithCasesDuplications",
			FrequencyMethod: priority_channels.ProbabilisticByCaseDuplication,
		},
		{
			Name:            "ProbabilisticByMultipleRandCalls",
			FrequencyMethod: priority_channels.ProbabilisticByMultipleRandCalls,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testProcessMessagesOfCombinedPriorityChannelsByFrequencyRatio_RandomTree(t,
				tc.FrequencyMethod)
		})
	}
}

func testProcessMessagesOfCombinedPriorityChannelsByFrequencyRatio_RandomTree(t *testing.T,
	frequencyMethod priority_channels.FrequencyMethod) {
	ctx, cancel := context.WithCancel(context.Background())

	priorityChannelsWithFreqRatio, channelsWithExpectedRatios := generateRandomPriorityChannelsTree(t, ctx, frequencyMethod)
	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, priorityChannelsWithFreqRatio, priority_channels.WithFrequencyMethod(frequencyMethod))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	messagesNum := 100000
	for channelName, cwr := range channelsWithExpectedRatios {
		go func(channelName string, cwr channelWithExpectedRatio) {
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
	for channelName, cwr := range channelsWithExpectedRatios {
		actualRatio := float64(countPerChannel[channelName]) / float64(totalCount)
		diff := math.Abs(cwr.expectedRatio - actualRatio)
		diffPercentage := (diff / cwr.expectedRatio) * 100
		if diffPercentage > 3 && frequencyMethod != priority_channels.ProbabilisticByCaseDuplication {
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

func generateRandomPriorityChannelsTree(t *testing.T, ctx context.Context, frequencyMethod priority_channels.FrequencyMethod) (
	[]priority_channels.PriorityChannelWithFreqRatio[string],
	map[string]channelWithExpectedRatio) {
	channelsWithExpectedRatios := make(map[string]channelWithExpectedRatio)
	// from 1 to 5 levels
	levelsNum := rand.N(3) + 1
	//levelsNum := 1
	// from 2 to 5 children per node
	childrenNum := rand.N(4) + 2
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

	priorityChannelsWithFreqRatio := make([]priority_channels.PriorityChannelWithFreqRatio[string], 0, childrenNum)
	for i := 0; i < childrenNum; i++ {
		childLabel := fmt.Sprintf("%d", i)
		channelName := fmt.Sprintf("priority-channel-%d-%s", levelsNum, childLabel)
		childChannel := priority_channels.NewPriorityChannelWithFreqRatio(
			channelName,
			generateRandomPriorityChannelsSubtree(t, ctx, levelsNum-1, childLabel, expectedRatios[i], frequencyMethod, channelsWithExpectedRatios),
			weights[i])
		priorityChannelsWithFreqRatio = append(priorityChannelsWithFreqRatio, childChannel)
	}
	sumOfAllChannels := 0.0
	for _, cwr := range channelsWithExpectedRatios {
		sumOfAllChannels += cwr.expectedRatio
	}
	if math.Abs(sumOfAllChannels-1.0) > 0.0001 {
		t.Fatalf("Expected sum of all priority channels to be %.4f, got %.4f\n", 1.0, sumOfAllChannels)
	}
	return priorityChannelsWithFreqRatio, channelsWithExpectedRatios
}

func generateRandomPriorityChannelsSubtree(t *testing.T, ctx context.Context,
	currLevel int, currLabel string, currExpectedRatio float64,
	frequencyMethod priority_channels.FrequencyMethod,
	channelsWithExpectedRatios map[string]channelWithExpectedRatio) *priority_channels.PriorityChannel[string] {
	var res *priority_channels.PriorityChannel[string]
	// from 1 to 5 children per node
	childrenNum := rand.N(5) + 1
	if childrenNum == 1 {
		cwr := channelWithExpectedRatio{
			channel:       make(chan string, 10),
			expectedRatio: currExpectedRatio,
		}
		childLabel := fmt.Sprintf("%s-%d", currLabel, 0)
		channelName := fmt.Sprintf("channel-%d-%s", currLevel, childLabel)
		if _, ok := channelsWithExpectedRatios[channelName]; ok {
			t.Fatalf("Channel %s: channel already exists\n", channelName)
		}
		channelsWithExpectedRatios[channelName] = cwr
		res, _ = priority_channels.WrapAsPriorityChannel(ctx, channelName, cwr.channel)
		return res
	}

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

	if currLevel == 0 {
		channelsWithFreqRatio := make([]channels.ChannelWithFreqRatio[string], 0, childrenNum)
		for i := 0; i < childrenNum; i++ {
			cwr := channelWithExpectedRatio{
				channel:       make(chan string, 10),
				expectedRatio: currExpectedRatio * expectedRatios[i],
			}
			childLabel := fmt.Sprintf("%s-%d", currLabel, i)
			channelName := fmt.Sprintf("channel-%d-%s", currLevel, childLabel)
			if _, ok := channelsWithExpectedRatios[channelName]; ok {
				t.Fatalf("Channel %s: channel already exists\n", channelName)
			}
			channelsWithExpectedRatios[channelName] = cwr
			childChannel := channels.NewChannelWithFreqRatio(channelName, cwr.channel, weights[i])
			channelsWithFreqRatio = append(channelsWithFreqRatio, childChannel)
		}
		res, _ = priority_channels.NewByFrequencyRatio[string](ctx, channelsWithFreqRatio)
	} else {
		priorityChannelsWithFreqRatio := make([]priority_channels.PriorityChannelWithFreqRatio[string], 0, childrenNum)
		for i := 0; i < childrenNum; i++ {
			childLabel := fmt.Sprintf("%s-%d", currLabel, i)
			channelName := fmt.Sprintf("priority-channel-%d-%s", currLevel, childLabel)
			childChannel := priority_channels.NewPriorityChannelWithFreqRatio(
				channelName,
				generateRandomPriorityChannelsSubtree(t, ctx, currLevel-1, childLabel, currExpectedRatio*expectedRatios[i], frequencyMethod, channelsWithExpectedRatios),
				weights[i])
			priorityChannelsWithFreqRatio = append(priorityChannelsWithFreqRatio, childChannel)
		}
		res, _ = priority_channels.CombineByFrequencyRatio[string](ctx, priorityChannelsWithFreqRatio, priority_channels.WithFrequencyMethod(frequencyMethod))
	}
	return res
}

func TestProcessMessagesByFreqRatioAmongFreqRatioChannelGroups_AutoDisableClosedChannels(t *testing.T) {
	ctx := context.Background()

	payingCustomerHighPriorityFlagshipProductC := make(chan string)
	payingCustomerHighPriorityNicheProductC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	inputChannels := []chan string{
		payingCustomerHighPriorityFlagshipProductC,
		payingCustomerHighPriorityNicheProductC,
		payingCustomerLowPriorityC,
		freeUserHighPriorityC,
		freeUserLowPriorityC,
	}

	for i := range inputChannels {
		go func(i int) {
			for j := 1; j <= 50; j++ {
				inputChannels[i] <- fmt.Sprintf("message %d", j)
			}
			close(inputChannels[i])
		}(i)
	}

	options := []func(*priority_channels.PriorityChannelOptions){
		priority_channels.AutoDisableClosedChannels(),
	}
	payingCustomerHighPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority - Flagship Product",
			payingCustomerHighPriorityFlagshipProductC,
			3),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority - Niche Product",
			payingCustomerHighPriorityNicheProductC,
			1),
	}, options...)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	payingCustomerLowPriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx,
		"Paying Customer - Low Priority", payingCustomerLowPriorityC)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	payingCustomerPriorityChannel, err := priority_channels.CombineByFrequencyRatio[string](ctx, []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityChannel,
			5),
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityChannel,
			1),
	}, options...)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			3),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	}, options...)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio("Paying Customer",
			payingCustomerPriorityChannel,
			6),
		priority_channels.NewPriorityChannelWithFreqRatio("Free User",
			freeUserPriorityChannel,
			4),
	}

	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio,
		priority_channels.AutoDisableClosedChannels())
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	receivedMessagesCount := 0
	for {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != priority_channels.ReceiveSuccess {
			if receivedMessagesCount != 250 {
				t.Errorf("Expected to receive 250 messages, but got %d", receivedMessagesCount)
			}
			if status != priority_channels.ReceiveNoOpenChannels {
				t.Errorf("Expected to receive 'no open channels' status on closure (%v), but got %v",
					priority_channels.ReceiveNoOpenChannels, status)
			}
			break
		}
		receivedMessagesCount++
		fmt.Printf("%s: %s\n", channelName, message)
		//time.Sleep(10 * time.Millisecond)
	}
}

func TestProcessMessagesScenario(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	channelsNum := 200
	allChannels := make([]chan string, channelsNum)
	for i := range allChannels {
		allChannels[i] = make(chan string)
	}
	allHighestPriorityFirstChannels := make([]channels.ChannelWithPriority[string], channelsNum)
	for i := range allChannels {
		allHighestPriorityFirstChannels[i] = channels.NewChannelWithPriority(
			fmt.Sprintf("Channel %d", i),
			allChannels[i],
			channelsNum-i)
	}
	freqRatio1Channel := make(chan string)

	// sending messages to individual channels
	var sg sync.WaitGroup
	sg.Add(2)
	go func() {
		for i := 1; i <= 100; i++ {
			allChannels[len(allChannels)-1] <- fmt.Sprintf("Freq-Ratio-9 - lowest priority message %d", i)
		}
		sg.Done()
	}()
	go func() {
		for i := 1; i <= 100; i++ {
			freqRatio1Channel <- fmt.Sprintf("Freq-Ratio-1 - message %d", i)
		}
		sg.Done()
	}()

	go func() {
		sg.Wait()
		cancel()
	}()

	freqRatio9PriorityChannel, err := priority_channels.NewByHighestAlwaysFirst(ctx, allHighestPriorityFirstChannels)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	freqRatio1PriorityChannel, err := priority_channels.WrapAsPriorityChannel(ctx, "Freq-Ratio-1", freqRatio1Channel)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Freq-Ratio-9",
			freqRatio9PriorityChannel,
			9),
		priority_channels.NewPriorityChannelWithFreqRatio(
			"Freq-Ratio-1", freqRatio1PriorityChannel, 1),
	}
	options := priority_channels.WithFrequencyMethod(priority_channels.StrictOrderFully)
	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio, options)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	results := make([]string, 0, 200)
	for {
		message, channelName, ok := ch.Receive()
		if !ok {
			break
		}
		fmt.Printf("%s\n", message)
		results = append(results, channelName)
	}
	if len(results) != 200 {
		t.Fatalf("Expected 200 results, but got %d", len(results))
	}

	for i := 1; i <= 110; i++ {
		if i%10 == 0 {
			if results[i-1] != "Freq-Ratio-1" {
				t.Errorf("Expected message %d to be from Channel 'Freq-Ratio-1', but got %s", i, results[i-1])
			}
		} else if results[i-1] != "Channel 199" {
			t.Errorf("Expected message %d to be from Channel 'Channel 199', but got %s", i, results[i-1])
		}
	}
	if results[110] != "Channel 199" {
		t.Errorf("Expected message %d to be from Channel 'Channel 199', but got %s", 111, results[110])
	}
	for i := 112; i <= 200; i++ {
		if results[i-1] != "Freq-Ratio-1" {
			t.Errorf("Expected message %d to be from Channel 'Freq-Ratio-1', but got %s", i, results[i-1])
		}
	}
}

func TestProcessMessagesByFreqRatioAmongFreqRatioChannelGroups_ChannelClosed(t *testing.T) {
	ctx := context.Background()
	payingCustomerHighPriorityC := make(chan string)
	payingCustomerLowPriorityC := make(chan string)
	freeUserHighPriorityC := make(chan string)
	freeUserLowPriorityC := make(chan string)

	payingCustomerPriorityChannel, err := priority_channels.NewByFrequencyRatio(ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Paying Customer - High Priority",
			payingCustomerHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Paying Customer - Low Priority",
			payingCustomerLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	freeUserPriorityChannel, err := priority_channels.NewByFrequencyRatio[string](ctx, []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio(
			"Free User - High Priority",
			freeUserHighPriorityC,
			5),
		channels.NewChannelWithFreqRatio(
			"Free User - Low Priority",
			freeUserLowPriorityC,
			1),
	})
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	channelsWithFreqRatio := []priority_channels.PriorityChannelWithFreqRatio[string]{
		priority_channels.NewPriorityChannelWithFreqRatio("Paying Customer",
			payingCustomerPriorityChannel,
			10),
		priority_channels.NewPriorityChannelWithFreqRatio("Free User",
			freeUserPriorityChannel,
			1),
	}
	ch, err := priority_channels.CombineByFrequencyRatio[string](ctx, channelsWithFreqRatio)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	close(freeUserHighPriorityC)

	for i := 0; i < 3; i++ {
		message, channelName, status := ch.ReceiveWithContext(context.Background())
		if status != priority_channels.ReceiveChannelClosed {
			t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", priority_channels.ReceiveChannelClosed, status)
		}
		if channelName != "Free User - High Priority" {
			t.Errorf("Expected channel name 'Free User - High Priority', but got %s", channelName)
		}
		if message != "" {
			t.Errorf("Expected empty message, but got %s", message)
		}
	}

	message, channelName, status := ch.ReceiveWithDefaultCase()
	if status != priority_channels.ReceiveChannelClosed {
		t.Errorf("Expected status ReceiveChannelClosed (%d), but got %d", priority_channels.ReceiveChannelClosed, status)
	}
	if channelName != "Free User - High Priority" {
		t.Errorf("Expected channel name 'Free User - High Priority', but got %s", channelName)
	}
	if message != "" {
		t.Errorf("Expected empty message, but got %s", message)
	}
}

func TestFreqRatioChannelGroupsValidation(t *testing.T) {
	var testCases = []struct {
		Name                   string
		ChannelsWithFreqRatios []channels.ChannelWithFreqRatio[string]
		ExpectedErrorMessage   string
	}{
		{
			Name:                   "No channels",
			ChannelsWithFreqRatios: []channels.ChannelWithFreqRatio[string]{},
			ExpectedErrorMessage:   priority_channels.ErrNoChannels.Error(),
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
			ExpectedErrorMessage: priority_channels.ErrEmptyChannelName.Error(),
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

			priorityChannels := make([]priority_channels.PriorityChannelWithFreqRatio[string], 0, len(tc.ChannelsWithFreqRatios))
			for _, ch := range tc.ChannelsWithFreqRatios {
				pch, err := priority_channels.WrapAsPriorityChannel(ctx, "******", make(chan string)) //ch.MsgsC())
				if err != nil {
					t.Fatalf("Unexpected error on wrapping as priority channel: %v", err)
				}
				priorityChannels = append(priorityChannels, priority_channels.NewPriorityChannelWithFreqRatio(
					ch.ChannelName(), pch, ch.FreqRatio()))
			}

			_, err := priority_channels.CombineByFrequencyRatio(ctx, priorityChannels)
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
