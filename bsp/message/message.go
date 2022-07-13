/*
	Message queue
*/
package message

type Message interface {
	Type() string
}

type Queue interface {
	Close() error
	Enqueue(msg Message) error
	PendingMessages() bool  // indicates that the queue has pending messages
	DiscardMessages() error // Drop all pending messages in the queue
	Messages() Iterator
}

type Iterator interface {
	// advances the Iterator. if no more messages or an error occure it returns false.
	Next() bool
	Message() Message
	Error() error
}

type QueueFactory func() Queue
