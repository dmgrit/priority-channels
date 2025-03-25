package priority_channels

import (
	"errors"
	"fmt"

	"github.com/dmgrit/priority-channels/strategies/frequency_strategies"
	"github.com/dmgrit/priority-channels/strategies/priority_strategies"
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
	maxChannelsForProbabilisticByCaseDuplication = 200
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

type Level int

const (
	LevelNew Level = iota
	LevelCombine
)

func getFrequencyStrategy(level Level, mode *FrequencyMode, method *FrequencyMethod, numChannels int) (PrioritizationStrategy[int], error) {
	frequencyMode := ProbabilisticMode
	if mode != nil {
		frequencyMode = *mode
	}
	if frequencyMode != StrictOrderMode && frequencyMode != ProbabilisticMode {
		return nil, ErrInvalidFrequencyMode
	}

	var frequencyMethod FrequencyMethod
	switch {
	case level == LevelNew && frequencyMode == ProbabilisticMode:
		if numChannels <= maxChannelsForProbabilisticByCaseDuplication {
			frequencyMethod = ProbabilisticByCaseDuplication
		} else {
			frequencyMethod = ProbabilisticByMultipleRandCalls
		}
	case level == LevelNew && frequencyMode == StrictOrderMode:
		frequencyMethod = StrictOrderAcrossCycles
	case level == LevelCombine && frequencyMode == ProbabilisticMode:
		frequencyMethod = ProbabilisticByMultipleRandCalls
	default:
		// level == LevelCombine && frequencyMode == StrictOrderMode:
		frequencyMethod = StrictOrderFully
	}

	if method != nil {
		// frequency method was explicitly set
		frequencyMethod = *method
	}

	switch frequencyMethod {
	case ProbabilisticByMultipleRandCalls:
		return priority_strategies.NewByProbabilityFromFreqRatios(), nil
	case ProbabilisticByCaseDuplication:
		if level == LevelCombine {
			return nil, &UnsupportedFrequencyMethodForCombineError{FrequencyMethod: frequencyMethod}
		}
		return frequency_strategies.NewProbabilisticByCaseDuplication(), nil
	case StrictOrderFully:
		return frequency_strategies.NewWithStrictOrderFully(), nil
	case StrictOrderAcrossCycles:
		if level == LevelCombine {
			return nil, &UnsupportedFrequencyMethodForCombineError{FrequencyMethod: frequencyMethod}
		}
		return frequency_strategies.NewWithStrictOrderAcrossCycles(), nil
	default:
		return nil, ErrInvalidFrequencyMethod
	}
}
