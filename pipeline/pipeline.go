package pipeline

import "context"

// Payload is implementaed by values that can be sent to the pipeline.
type Payload interface {
	// Clone returns a new Payload that's a deep-copy of the original.
	Clone() Payload

	// MarkAsProcessed called by the pipeline when it reaches the output sink
	// or discarded by the pipeline.
	MarkAsProcessed()
}

// Processor is implemented by types that can process the Payload as part of
// pipeline stage.
type Processor interface {
	// Process takes the input Payload and return a new Payload to be sent either to the
	// next stage or the output sink. Process can also apt to prevent the Payload from reaching
	// the next stage by returning nil.
	Process(context.Context, Payload) (Payload, error)
}

// ProcessorFunc is adapter to Process
type ProcessorFunc func(context.Context, Payload) (Payload, error)

func (f ProcessorFunc) Process(ctx context.Context, p Payload) (Payload, error) {
	return f(ctx, p)
}

// StageParams includes the information required for execute of a pipeline stage.
// StageParams instance is passed to the Run() method of each stage.

type StageParams interface {
	// StageIndex returns the position of a stage in the pipeline.
	StageIndex() int
	// Input returns a channel for reading the input Payload into the stage.
	Input() <-chan Payload
	// Output returns a channel for writing the stage output.
	Output() chan<- Payload
	// Error returns a channel for writing the errors that were encountered
	// during the stage execution.
	Error() chan<- error
}

// StageRunner implemented by types that can be chained together to form multi-stage pipeline.
type StageRunner interface {
	// Run implement the process logic of a stage. Run reads input payload from Input channel
	// and writes its output to Output channel.
	// Calls to Run expected to block until one of the following occurs:
	// - Input channel is closed.
	// - Its context got cancelled
	// - Error happen while processing the payload.
	Run(context.Context, StageParams)
}

type Source interface {
	// Next fetches the next Payload. If an error occur or no more payload
	// it returns false
	Next(context.Context) error

	// Payload returns a Payload to be processed
	Payload() Payload

	// Error return the last error observed by the source.
	Error() error
}

type Sink interface {
	// Consume process a Payload instance that's outputted by the pipeline.
	Consume(context.Context, Payload)
}
