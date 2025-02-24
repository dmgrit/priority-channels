package strategies

import "fmt"

type WeightValidationError struct {
	ChannelIndex int
	Err          error
}

func (e *WeightValidationError) Error() string {
	return fmt.Sprintf("channel %d: %v", e.ChannelIndex, e.Err)
}

func convertWeightsWithTypeAssertion[W any](weightsLabel string, weights []interface{}) ([]W, error) {
	result := make([]W, 0, len(weights))
	for i, v := range weights {
		w, ok := v.(W)
		if !ok {
			return nil, &WeightValidationError{
				ChannelIndex: i,
				Err:          fmt.Errorf("%s must be of type %T", weightsLabel, w),
			}
		}
		result = append(result, w)
	}
	return result, nil
}
