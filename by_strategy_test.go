package priority_channels_test

import (
	"context"
	"errors"
	"fmt"
	pc "github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
	"sort"
	"testing"
	"time"
)

func TestProcessMessagesByStrategy(t *testing.T) {
	msgsChannels := make([]chan *Msg, 3)
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

func TestProcessMessagesByDynamicStrategy(t *testing.T) {
	msgsChannels := make([]chan *Msg, 3)
	msgsChannels[0] = make(chan *Msg, 5)
	msgsChannels[1] = make(chan *Msg, 5)
	msgsChannels[2] = make(chan *Msg, 5)

	strategiesByName := map[string]strategies.DynamicSubStrategy{
		"DayTime":   frequency_strategies.NewWithStrictOrderFully(),
		"NightTime": newByFirstDecimalDigitAsc(),
	}
	channels := []channels.ChannelWithWeight[*Msg, map[string]interface{}]{
		channels.NewChannelWithWeight("Freq-1-First-Decimal-Digit-3", msgsChannels[0],
			map[string]interface{}{
				"DayTime":   1,
				"NightTime": 1.3,
			}),
		channels.NewChannelWithWeight("Freq-2-First-Decimal-Digit-1", msgsChannels[1],
			map[string]interface{}{
				"DayTime":   2,
				"NightTime": 2.1,
			}),
		channels.NewChannelWithWeight("Freq-3-First-Decimal-Digit-2", msgsChannels[2],
			map[string]interface{}{
				"DayTime":   3,
				"NightTime": 3.2,
			}),
	}

	var hour int
	currentStrategySelector := func() string {
		if 7 <= hour && hour <= 18 {
			return "DayTime"
		} else {
			return "NightTime"
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := pc.NewByStrategy(ctx,
		strategies.NewDynamic(strategiesByName, currentStrategySelector),
		channels,
	)
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	var strategiesCases = []struct {
		strategyName    string
		hour            int
		ExpectedResults []*Msg
	}{
		{
			strategyName: "DayTime - FreqRatio",
			hour:         10,
			ExpectedResults: []*Msg{
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-1"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-2"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-3"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-1"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-2"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-1"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-4"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-5"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-3"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-4"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-2"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-5"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-3"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-4"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-5"},
			},
		},
		{
			strategyName: "NightTime - FirstDecimalDigit",
			hour:         20,
			ExpectedResults: []*Msg{
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-1"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-2"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-3"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-4"},
				{Body: "Freq-2-First-Decimal-Digit-1 Msg-5"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-1"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-2"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-3"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-4"},
				{Body: "Freq-3-First-Decimal-Digit-2 Msg-5"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-1"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-2"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-3"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-4"},
				{Body: "Freq-1-First-Decimal-Digit-3 Msg-5"},
			},
		},
	}

	for _, sc := range strategiesCases {
		hour = sc.hour

		// send 15 messages to the channels
		for i := 0; i <= 2; i++ {
			for j := 1; j <= 5; j++ {
				msgsChannels[i] <- &Msg{Body: fmt.Sprintf("%s Msg-%d", channels[i].ChannelName(), j)}
			}

		}

		var results []*Msg
		for i := 1; i <= 15; i++ {
			msg, _, ok := ch.Receive()
			if !ok {
				t.Fatalf("Unexpected error on receiving message from priority channel")
			}
			results = append(results, msg)
		}

		if len(results) != len(sc.ExpectedResults) {
			t.Errorf("%s: Expected %d results, but got %d", sc.strategyName, len(sc.ExpectedResults), len(results))
		}
		for i := range results {
			if results[i].Body != sc.ExpectedResults[i].Body {
				t.Errorf("%s: Result %d: Expected message %s, but got %s",
					sc.strategyName, i, sc.ExpectedResults[i].Body, results[i].Body)
			}
		}
	}

	cancel()
}

func TestProcessMessagesByDynamicStrategy_TypeAssertion(t *testing.T) {
	msgsChannels := make([]chan *Msg, 3)
	msgsChannels[0] = make(chan *Msg, 5)
	msgsChannels[1] = make(chan *Msg, 5)

	var testCases = []struct {
		Name                 string
		Strategy             strategies.DynamicSubStrategy
		InvalidWeight        interface{}
		InvalidStrategies    map[string]interface{}
		ExpectedErrorMessage string
	}{
		{
			Name:                 "ByFreqRatio",
			Strategy:             frequency_strategies.NewWithStrictOrderAcrossCycles(),
			InvalidWeight:        1.3,
			ExpectedErrorMessage: "channel 'Channel 1': frequency ratio must be of type int",
		},
		{
			Name:                 "ByHighestAlwaysFirst",
			Strategy:             strategies.NewByHighestAlwaysFirst(),
			InvalidWeight:        1.3,
			ExpectedErrorMessage: "channel 'Channel 1': priority must be of type int",
		},
		{
			Name:                 "ByProbability",
			Strategy:             strategies.NewByProbability(),
			InvalidWeight:        1,
			ExpectedErrorMessage: "channel 'Channel 1': probability must be of type float64",
		},
		{
			Name:                 "ByFirstDecimalDigit",
			Strategy:             newByFirstDecimalDigitAsc(),
			InvalidWeight:        1,
			ExpectedErrorMessage: "channel 'Channel 1': first-decimal-digit priority must be of type float64",
		},
		{
			Name:                 "Invalid number of strategies",
			Strategy:             frequency_strategies.NewWithStrictOrderAcrossCycles(),
			InvalidStrategies:    map[string]interface{}{"DayTime": 1},
			ExpectedErrorMessage: "channel 'Channel 1': invalid number of strategies: 1, expected 2",
		},
		{
			Name:                 "Unknown strategy",
			Strategy:             frequency_strategies.NewWithStrictOrderAcrossCycles(),
			InvalidStrategies:    map[string]interface{}{"DayTime": 1, "PartyTime": 10},
			ExpectedErrorMessage: "channel 'Channel 1': unknown strategy PartyTime",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("SubStrategyTypeAssertion_%s", tc.Name), func(t *testing.T) {
			strategiesByName := map[string]strategies.DynamicSubStrategy{
				"DayTime":   tc.Strategy,
				"NightTime": newByFirstDecimalDigitAsc(),
			}

			var channelWithInvalidWeights channels.ChannelWithWeight[*Msg, map[string]interface{}]
			if tc.InvalidStrategies != nil {
				channelWithInvalidWeights = channels.NewChannelWithWeight("Channel 1", msgsChannels[0], tc.InvalidStrategies)
			} else {
				channelWithInvalidWeights = channels.NewChannelWithWeight("Channel 1", msgsChannels[0], map[string]interface{}{
					"DayTime":   tc.InvalidWeight,
					"NightTime": 1.3,
				})
			}

			channels := []channels.ChannelWithWeight[*Msg, map[string]interface{}]{
				channelWithInvalidWeights,
				channels.NewChannelWithWeight("Channel 2", msgsChannels[1],
					map[string]interface{}{
						"DayTime":   2,
						"NightTime": 2.1,
					}),
			}

			currentStrategySelector := func() string { return "DayTime" }

			_, err := pc.NewByStrategy(context.Background(),
				strategies.NewDynamic(strategiesByName, currentStrategySelector),
				channels,
			)
			if err == nil {
				t.Fatalf("Expected error on priority channel intialization, but got nil")
			}
			if err.Error() != tc.ExpectedErrorMessage {
				t.Errorf("Expected error message '%s', but got '%s'", tc.ExpectedErrorMessage, err.Error())
			}
		})
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

func (s *byFirstDecimalDigit) InitializeWithTypeAssertion(priorities []interface{}) error {
	prioritiesFloat64 := make([]float64, 0, len(priorities))
	for i, priority := range priorities {
		priorityFloat64, ok := priority.(float64)
		if !ok {
			return &strategies.WeightValidationError{
				ChannelIndex: i,
				Err:          errors.New("first-decimal-digit priority must be of type float64"),
			}
		}
		prioritiesFloat64 = append(prioritiesFloat64, priorityFloat64)
	}
	return s.Initialize(prioritiesFloat64)
}

func (s *byFirstDecimalDigit) NextSelectCasesRankedIndexes(upto int) ([]strategies.RankedIndex, bool) {
	res := make([]strategies.RankedIndex, 0, upto)
	for i := 0; i < upto && i < len(s.sortedPriorities); i++ {
		res = append(res, strategies.RankedIndex{Index: s.sortedPriorities[i].OriginalIndex, Rank: i})
	}
	return res, len(res) == len(s.sortedPriorities)
}

func (s *byFirstDecimalDigit) UpdateOnCaseSelected(index int) {}

func (s *byFirstDecimalDigit) DisableSelectCase(index int) {
	// to be implemented
}
