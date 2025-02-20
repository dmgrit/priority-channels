package strategies

import "fmt"

type WeightValidationError struct {
	ChannelIndex int
	Err          error
}

func (e *WeightValidationError) Error() string {
	return fmt.Sprintf("channel %d: %v", e.ChannelIndex, e.Err)
}
