package priority_channels

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dmgrit/priority-channels/strategies"
	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

func TestGetFrequencyStrategy(t *testing.T) {
	probabilisticMode := ProbabilisticMode
	strictOrderMode := StrictOrderMode
	probabilisticByCaseDuplication := ProbabilisticByCaseDuplication
	probabilisticByMultipleRandCalls := ProbabilisticByMultipleRandCalls
	strictOrderAcrossCycles := StrictOrderAcrossCycles
	strictOrderFully := StrictOrderFully
	unsupportedMode := FrequencyMode(1000)
	unsupportedMethod := FrequencyMethod(1000)

	var testCases = []struct {
		name             string
		level            priorityChannelLevel
		mode             *FrequencyMode
		method           *FrequencyMethod
		numChannels      int
		expectedStrategy strategies.PrioritizationStrategy[int]
		expectedError    error
	}{
		{
			name:          "NewLevel - Unsupported Mode - Expected Error",
			level:         levelNew,
			mode:          &unsupportedMode,
			numChannels:   10,
			expectedError: ErrInvalidFrequencyMode,
		},
		{
			name:          "NewLevel - Unsupported Method - Expected Error",
			level:         levelNew,
			method:        &unsupportedMethod,
			numChannels:   10,
			expectedError: ErrInvalidFrequencyMethod,
		},
		{
			name:             "NewLevel - Default strategy is Strict Order Across Cycles",
			level:            levelNew,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewWithStrictOrderAcrossCycles(),
		},
		{
			name:             "NewLevel - If By Probability method is given it is used",
			level:            levelNew,
			numChannels:      10,
			method:           &probabilisticByMultipleRandCalls,
			expectedStrategy: frequency_strategies.NewByProbabilityFromFreqRatios(),
		},
		{
			name:             "NewLevel - Default for Probabilistic Mode is By Case Duplication",
			level:            levelNew,
			mode:             &probabilisticMode,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewProbabilisticByCaseDuplication(),
		},
		{
			name:             "NewLevel - Probability Mode - If number of channels is greater than recommended - strategy is By Probability",
			level:            levelNew,
			mode:             &probabilisticMode,
			numChannels:      maxRecommendedChannelsForCaseDuplication + 1,
			expectedStrategy: frequency_strategies.NewByProbabilityFromFreqRatios(),
		},
		{
			name:             "NewLevel - Strategy provided but number of channels is greater than recommended - still use provided Strategy",
			level:            levelNew,
			numChannels:      maxRecommendedChannelsForCaseDuplication + 1,
			method:           &probabilisticByCaseDuplication,
			expectedStrategy: frequency_strategies.NewProbabilisticByCaseDuplication(),
		},
		{
			name:        "NewLevel - Strategy is Case Duplication and number of channels is greater than supported - Expected Error",
			level:       levelNew,
			method:      &probabilisticByCaseDuplication,
			numChannels: maxSupportedChannelsForCaseDuplication + 1,
			expectedError: fmt.Errorf("too many channels %d for frequency method %s (max %d)",
				maxSupportedChannelsForCaseDuplication+1, frequencyMethodNames[ProbabilisticByCaseDuplication], maxSupportedChannelsForCaseDuplication),
		},
		{
			name:             "NewLevel - Default for strict mode is Strict Order Across Cycles",
			level:            levelNew,
			mode:             &strictOrderMode,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewWithStrictOrderAcrossCycles(),
		},
		{
			name:             "NewLevel - if Strict Order Fully method is provided it is used",
			level:            levelNew,
			method:           &strictOrderFully,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewWithStrictOrderFully(),
		},
		{
			name:             "CombineLevel - Default strategy is Strict Order Fully",
			level:            levelCombine,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewWithStrictOrderFully(),
		},
		{
			name:             "CombineLevel - Strategy for probabilistic mode is By Probability",
			level:            levelCombine,
			mode:             &probabilisticMode,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewByProbabilityFromFreqRatios(),
		},
		{
			name:          "CombineLevel - Case Duplication method is not supported",
			level:         levelCombine,
			method:        &probabilisticByCaseDuplication,
			numChannels:   10,
			expectedError: &UnsupportedFrequencyMethodForCombineError{FrequencyMethod: ProbabilisticByCaseDuplication},
		},
		{
			name:             "CombineLevel - Strategy for strict mode is Strict Order Fully",
			level:            levelCombine,
			mode:             &strictOrderMode,
			numChannels:      10,
			expectedStrategy: frequency_strategies.NewWithStrictOrderFully(),
		},
		{
			name:          "CombineLevel - Strict Order Across Cycles method is not supported",
			level:         levelCombine,
			method:        &strictOrderAcrossCycles,
			numChannels:   10,
			expectedError: &UnsupportedFrequencyMethodForCombineError{FrequencyMethod: StrictOrderAcrossCycles},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := getFrequencyStrategy(tc.level, tc.mode, tc.method, tc.numChannels)
			if tc.expectedError != nil {
				if err == nil || err.Error() != tc.expectedError.Error() {
					t.Errorf("Expected error %v, got %v", tc.expectedError, err)
				}
				return
			}
			if reflect.TypeOf(res) != reflect.TypeOf(tc.expectedStrategy) {
				t.Errorf("Expected strategy type %v, got %v", reflect.TypeOf(tc.expectedStrategy), reflect.TypeOf(res))
			}
		})
	}
}
