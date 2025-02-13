package priority_channels_test

import (
	"context"
	"testing"

	priority_channels "github.com/dmgrit/priority-channels"
)

func TestWrapAsPriorityChannelValidation(t *testing.T) {
	_, err := priority_channels.WrapAsPriorityChannel(
		context.Background(), "", make(chan string))
	if err == nil {
		t.Fatalf("Expected validation error")
	}

	expectedErrorMessage := priority_channels.ErrEmptyChannelName.Error()
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error %s, but got: %v", expectedErrorMessage, err)
	}
}
