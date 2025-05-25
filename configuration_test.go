package priority_channels_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dmgrit/priority-channels"
)

func TestNewFromConfiguration(t *testing.T) {
	config := priority_channels.Configuration{
		PriorityChannel: &priority_channels.PriorityChannelConfig{
			Method: "by-frequency-ratio",
			Channels: []priority_channels.ChannelConfig{
				{
					Name:      "channel-1",
					FreqRatio: 2,
				},
				{
					Name:      "channel-2",
					FreqRatio: 3,
				},
				{
					Name:      "priority-channel-3",
					FreqRatio: 3,
					PriorityChannelConfig: &priority_channels.PriorityChannelConfig{
						Method: "by-highest-always-first",
						Channels: []priority_channels.ChannelConfig{
							{
								Name:     "channel-3",
								Priority: 1,
							},
							{
								Name:     "channel-4",
								Priority: 2,
							},
						},
					},
				},
			},
		},
	}

	// Call the function to create a new channel from the configuration
	var channelNameToChannel = map[string]<-chan string{
		"channel-1": make(chan string),
		"channel-2": make(chan string),
		"channel-3": make(chan string),
		"channel-4": make(chan string),
	}

	channel, err := priority_channels.NewFromConfiguration(context.Background(), config, channelNameToChannel, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if channel == nil {
		t.Fatalf("expected a channel, got nil")
	}

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	t.Logf(string(jsonConfig))
}
