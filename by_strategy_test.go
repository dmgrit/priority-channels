package priority_channels_test

import (
	"context"
	"fmt"
	pc "github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies"
	"sort"
	"testing"
	"time"
)

func TestProcessMessagesByStrategy(t *testing.T) {
	msgsChannels := make([]chan *Msg, 4)
	msgsChannels[0] = make(chan *Msg, 5)
	msgsChannels[1] = make(chan *Msg, 5)
	msgsChannels[2] = make(chan *Msg, 5)

	channels := []channels.ChannelWithWeight[*Msg, float64]{
		channels.NewChannelWithWeight("First-Decimal-Digit-3", msgsChannels[0], 1.3),
		channels.NewChannelWithWeight("First-Decimal-Digit-1", msgsChannels[1], 2.1),
		channels.NewChannelWithWeight("First-Decimal-Digit-2", msgsChannels[2], 3.2),
	}

	for i := 0; i <= 2; i++ {
		for j := 1; j <= 5; j++ {
			msgsChannels[i] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[i].ChannelName(), j)}
		}
	}

	var results []*Msg
	msgProcessor := func(_ context.Context, msg *Msg, channelName string) {
		results = append(results, msg)
	}
	ctx, cancel := context.WithCancel(context.Background())

	strategy := newByFirstDecimalDigitAsc()
	priorityChannel, err := pc.NewByStrategy[*Msg, float64](ctx, strategy, channels)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}
	go pc.ProcessPriorityChannelMessages(priorityChannel, msgProcessor)

	time.Sleep(3 * time.Second)
	cancel()

	expectedResults := []*Msg{
		{Body: "First-Decimal-Digit-1 Msg-1"},
		{Body: "First-Decimal-Digit-1 Msg-2"},
		{Body: "First-Decimal-Digit-1 Msg-3"},
		{Body: "First-Decimal-Digit-1 Msg-4"},
		{Body: "First-Decimal-Digit-1 Msg-5"},
		{Body: "First-Decimal-Digit-2 Msg-1"},
		{Body: "First-Decimal-Digit-2 Msg-2"},
		{Body: "First-Decimal-Digit-2 Msg-3"},
		{Body: "First-Decimal-Digit-2 Msg-4"},
		{Body: "First-Decimal-Digit-2 Msg-5"},
		{Body: "First-Decimal-Digit-3 Msg-1"},
		{Body: "First-Decimal-Digit-3 Msg-2"},
		{Body: "First-Decimal-Digit-3 Msg-3"},
		{Body: "First-Decimal-Digit-3 Msg-4"},
		{Body: "First-Decimal-Digit-3 Msg-5"},
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

type byFirstDecimalDigit struct {
	sortedPriorities []sortedToOriginalIndex
}

type sortedToOriginalIndex struct {
	FirstDecimalDigit int
	OriginalIndex     int
}

func newByFirstDecimalDigitAsc() *byFirstDecimalDigit {
	return &byFirstDecimalDigit{}
}

func (s *byFirstDecimalDigit) Initialize(priorities []float64) error {
	s.sortedPriorities = make([]sortedToOriginalIndex, 0, len(priorities))
	for i, p := range priorities {
		if p < 0 {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          strategies.ErrPriorityIsNegative,
			}
		}
		firstDecimalDigit := int(p*10) % 10
		s.sortedPriorities = append(s.sortedPriorities, sortedToOriginalIndex{
			FirstDecimalDigit: firstDecimalDigit,
			OriginalIndex:     i,
		})
	}
	sort.Slice(s.sortedPriorities, func(i, j int) bool {
		return s.sortedPriorities[i].FirstDecimalDigit < s.sortedPriorities[j].FirstDecimalDigit
	})
	return nil
}

func (s *byFirstDecimalDigit) NextSelectCasesIndexes(upto int) []int {
	res := make([]int, 0, upto)
	for i := 0; i < upto && i < len(s.sortedPriorities); i++ {
		res = append(res, s.sortedPriorities[i].OriginalIndex)
	}
	return res
}

func (s *byFirstDecimalDigit) UpdateOnCaseSelected(index int) {}
