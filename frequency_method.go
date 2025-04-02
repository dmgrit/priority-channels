package priority_channels

import (
	"errors"
	"fmt"

	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
)

var (
	ErrInvalidFrequencyMode   = errors.New("invalid frequency mode")
	ErrInvalidFrequencyMethod = errors.New("invalid frequency method")
)

type UnsupportedFrequencyMethodForCombineError struct {
	FrequencyMethod FrequencyMethod
}

func (e *UnsupportedFrequencyMethodForCombineError) Error() string {
	return fmt.Sprintf("frequency method %v does not support combining priority channels", frequencyMethodNames[e.FrequencyMethod])
}

const (
	maxRecommendedChannelsForCaseDuplication = 200
	// 65536 is maximum supported number of channels in select statement, and 3 channels are reserved for internal use
	maxSupportedChannelsForCaseDuplication = 65533
)

type FrequencyMethod int

const (
	StrictOrderAcrossCycles FrequencyMethod = iota
	StrictOrderFully
	ProbabilisticByCaseDuplication
	ProbabilisticByMultipleRandCalls
)

var frequencyMethodNames = map[FrequencyMethod]string{
	StrictOrderAcrossCycles:          "StrictOrderAcrossCycles",
	StrictOrderFully:                 "StrictOrderFully",
	ProbabilisticByCaseDuplication:   "ProbabilisticByCaseDuplication",
	ProbabilisticByMultipleRandCalls: "ProbabilisticByMultipleRandCalls",
}

type FrequencyMode int

const (
	StrictOrderMode FrequencyMode = iota
	ProbabilisticMode
)

var frequencyModeNames = map[FrequencyMode]string{
	StrictOrderMode:   "StrictOrderMode",
	ProbabilisticMode: "ProbabilisticMode",
}

type priorityChannelLevel int

const (
	levelNew priorityChannelLevel = iota
	levelCombine
)

func getFrequencyStrategy(level priorityChannelLevel, mode *FrequencyMode, method *FrequencyMethod, numChannels int) (PrioritizationStrategy[int], error) {
	frequencyMode := ProbabilisticMode
	if mode != nil {
		frequencyMode = *mode
	}
	if frequencyMode != StrictOrderMode && frequencyMode != ProbabilisticMode {
		return nil, ErrInvalidFrequencyMode
	}

	var frequencyMethod FrequencyMethod
	switch {
	case level == levelNew && frequencyMode == ProbabilisticMode:
		if numChannels <= maxRecommendedChannelsForCaseDuplication {
			frequencyMethod = ProbabilisticByCaseDuplication
		} else {
			frequencyMethod = ProbabilisticByMultipleRandCalls
		}
	case level == levelNew && frequencyMode == StrictOrderMode:
		frequencyMethod = StrictOrderAcrossCycles
	case level == levelCombine && frequencyMode == ProbabilisticMode:
		frequencyMethod = ProbabilisticByMultipleRandCalls
	default:
		// level == levelCombine && frequencyMode == StrictOrderMode:
		frequencyMethod = StrictOrderFully
	}

	if method != nil {
		// frequency method was explicitly set
		frequencyMethod = *method
	}

	switch frequencyMethod {
	case ProbabilisticByMultipleRandCalls:
		return frequency_strategies.NewByProbabilityFromFreqRatios(), nil
	case ProbabilisticByCaseDuplication:
		if level == levelCombine {
			return nil, &UnsupportedFrequencyMethodForCombineError{FrequencyMethod: frequencyMethod}
		}
		if numChannels > maxSupportedChannelsForCaseDuplication {
			return nil, fmt.Errorf("too many channels %d for frequency method %s (max %d)",
				numChannels, frequencyMethodNames[frequencyMethod], maxSupportedChannelsForCaseDuplication)
		}
		return frequency_strategies.NewProbabilisticByCaseDuplication(), nil
	case StrictOrderFully:
		return frequency_strategies.NewWithStrictOrderFully(), nil
	case StrictOrderAcrossCycles:
		if level == levelCombine {
			return nil, &UnsupportedFrequencyMethodForCombineError{FrequencyMethod: frequencyMethod}
		}
		return frequency_strategies.NewWithStrictOrderAcrossCycles(), nil
	default:
		return nil, ErrInvalidFrequencyMethod
	}
}
