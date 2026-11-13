package gamecore

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	queueCapacity = 8192
	batchSize     = 256
	flushInterval = 10 * time.Millisecond
	syncPoolSize  = 8
)

// Bridge mirrors semantic events asynchronously and serves latency-sensitive
// gameplay decisions over a bounded pool of persistent IPC connections.
type Bridge struct {
	client         *Client
	syncClients    []*Client
	nextSyncClient atomic.Uint64
	queue          chan Event
	stop           chan struct{}
	done           chan struct{}
	once           sync.Once
	onError        func(error)
}

func NewBridge(socket string, onError func(error)) *Bridge {
	syncClients := make([]*Client, syncPoolSize)
	for i := range syncClients {
		syncClients[i] = &Client{SocketPath: socket, Timeout: 50 * time.Millisecond}
	}
	b := &Bridge{
		client:     &Client{SocketPath: socket, Timeout: 500 * time.Millisecond},
		syncClients: syncClients,
		queue:      make(chan Event, queueCapacity),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		onError:    onError,
	}
	go b.run()
	return b
}

// Enqueue never waits for the Rust process. False means the bridge is closed
// or its bounded queue is full; callers must continue normal Go gameplay.
func (b *Bridge) Enqueue(event Event) bool {
	if b == nil {
		return false
	}
	select {
	case <-b.stop:
		return false
	default:
	}
	event.ProtocolVersion = ProtocolVersion
	select {
	case b.queue <- event:
		return true
	case <-b.stop:
		return false
	default:
		b.report(fmt.Errorf("rust game-core event queue is full; dropping %s event", event.Type))
		return false
	}
}

// Call sends a request through a persistent IPC connection and waits for its
// result. A small connection pool prevents one player's mining request from
// serializing every other player's tick behind the same Go mutex. Rust still
// serializes state mutations under its authoritative CoreState lock.
func (b *Bridge) Call(ctx context.Context, event Event) (Response, error) {
	if b == nil || len(b.syncClients) == 0 {
		return Response{}, fmt.Errorf("rust game-core synchronous client is unavailable")
	}
	index := b.nextSyncClient.Add(1) - 1
	client := b.syncClients[index%uint64(len(b.syncClients))]
	return client.Call(ctx, event)
}

func (b *Bridge) Close() {
	if b == nil {
		return
	}
	b.once.Do(func() { close(b.stop) })
	<-b.done
	_ = b.client.Close()
	for _, client := range b.syncClients {
		if client != nil {
			_ = client.Close()
		}
	}
}

func (b *Bridge) report(err error) {
	if b.onError != nil {
		b.onError(err)
	}
}

func (b *Bridge) run() {
	defer close(b.done)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	batch := make([]Event, 0, batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
		defer cancel()
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			_, err = b.client.CallBatch(ctx, batch)
			if err == nil {
				batch = batch[:0]
				return
			}
			select {
			case <-ctx.Done():
				attempt = 3
			case <-time.After(time.Duration(attempt+1) * 50 * time.Millisecond):
			}
		}
		b.report(fmt.Errorf("rust game-core batch (%d events) not delivered: %w", len(batch), err))
		batch = batch[:0]
	}

	for {
		select {
		case event := <-b.queue:
			batch = append(batch, event)
			if len(batch) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-b.stop:
			for {
				select {
				case event := <-b.queue:
					batch = append(batch, event)
					if len(batch) >= batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		}
	}
}
