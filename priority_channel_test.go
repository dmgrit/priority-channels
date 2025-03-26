package priority_channels_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dmgrit/priority-channels"
	"github.com/dmgrit/priority-channels/channels"
)

func TestReceiveWithDefaultCaseSynchronized(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var inputChannels = []chan string{
		make(chan string),
		make(chan string),
	}
	channelsWithFreqRatio := []channels.ChannelWithFreqRatio[string]{
		channels.NewChannelWithFreqRatio("Channel A", inputChannels[0], 1),
		channels.NewChannelWithFreqRatio("Channel B", inputChannels[1], 2),
	}

	ch, err := priority_channels.NewByFrequencyRatio(ctx, channelsWithFreqRatio, priority_channels.Synchronized(true))
	if err != nil {
		t.Fatalf("Unexpected error on priority channel intialization: %v", err)
	}

	m := make(map[priority_channels.ReceiveStatus]int)
	var mtx sync.Mutex

	receiveWithContextDone := make(chan struct{})
	go func() {
		_, _, status := ch.ReceiveWithContext(ctx)
		mtx.Lock()
		m[status]++
		mtx.Unlock()
		close(receiveWithContextDone)
	}()

	time.Sleep(1 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, status := ch.ReceiveWithDefaultCase()
			mtx.Lock()
			m[status]++
			mtx.Unlock()
		}(i)
	}
	wg.Wait()

	select {
	case <-receiveWithContextDone:
		t.Fatalf("Expected ReceiveWithContext not to finish running before message is received")
	default:
	}

	mtx.Lock()
	if !reflect.DeepEqual(m, map[priority_channels.ReceiveStatus]int{
		priority_channels.ReceiveDefaultCase: 10,
	}) {
		t.Fatalf("Expected 10 received with default case, got %v", m)
	}
	mtx.Unlock()

	inputChannels[0] <- "test"
	select {
	case <-time.After(1 * time.Millisecond):
		t.Fatalf("Expected message to be received")
	case <-receiveWithContextDone:
	}

	if !reflect.DeepEqual(m, map[priority_channels.ReceiveStatus]int{
		priority_channels.ReceiveDefaultCase: 10,
		priority_channels.ReceiveSuccess:     1,
	}) {
		t.Fatalf("Expected 10 exit on default case and 1 received successfully, got %v", m)
	}
}
