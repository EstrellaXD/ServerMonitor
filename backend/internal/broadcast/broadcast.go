package broadcast

import (
	"log/slog"
	"sync"
	"sync/atomic"
)

const subscriberBuffer = 64

// Broadcaster fans out byte messages to many subscribers. Slow consumers are
// skipped (matching tokio::sync::broadcast lag-tolerant semantics).
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan []byte
	nextID      atomic.Uint64
}

func New() *Broadcaster {
	return &Broadcaster{subscribers: make(map[uint64]chan []byte)}
}

// Subscribe returns a receive-only channel and an unsubscribe func that the
// caller MUST invoke to free the slot.
func (b *Broadcaster) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, subscriberBuffer)
	id := b.nextID.Add(1)

	b.mu.Lock()
	b.subscribers[id] = ch
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		if existing, ok := b.subscribers[id]; ok && existing == ch {
			delete(b.subscribers, id)
			close(ch)
		}
		b.mu.Unlock()
	}
}

func (b *Broadcaster) Publish(msg []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for id, ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			slog.Warn("dropping broadcast frame for slow subscriber", "id", id)
		}
	}
}

func (b *Broadcaster) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
