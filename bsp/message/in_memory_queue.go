package message

import "sync"

var _ Queue = (*inMemoryQueue)(nil)

type inMemoryQueue struct {
	mu         sync.Mutex
	msgs       []Message
	latchedMsg Message
}

// inMemoryQueue is concurrency safe for enqueue and dequeue but the returned iterator is not safe for concurrent access.
func NewInMemoryQueue() *inMemoryQueue {
	return new(inMemoryQueue)
}

func (q *inMemoryQueue) Enqueue(msg Message) error {
	defer q.mu.Unlock()
	q.mu.Lock()
	q.msgs = append(q.msgs, msg)
	return nil
}

func (q *inMemoryQueue) PendingMessages() bool {
	return len(q.msgs) != 0
}

func (q *inMemoryQueue) DiscardMessages() error {
	defer q.mu.Unlock()
	q.mu.Lock()
	q.msgs = q.msgs[:0]
	return nil
}

func (*inMemoryQueue) Close() error { return nil }

func (q *inMemoryQueue) Messages() Iterator {
	return q
}

func (q *inMemoryQueue) Next() bool {
	defer q.mu.Unlock()
	q.mu.Lock()
	qLen := len(q.msgs)
	if qLen == 0 {
		return false
	}

	// Dequeue message from the tail of the queue.
	q.latchedMsg = q.msgs[qLen-1]
	q.msgs = q.msgs[:qLen-1]
	return true
}

func (q *inMemoryQueue) Message() Message {
	q.mu.Lock()
	msg := q.latchedMsg
	q.mu.Unlock()
	return msg
}

func (*inMemoryQueue) Error() error { return nil }
