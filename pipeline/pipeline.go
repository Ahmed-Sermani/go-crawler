package pipeline

import (
	"context"
	"sync"

	"github.com/hashicorp/go-multierror"
)

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
	Next(context.Context) bool

	// Payload returns a Payload to be processed
	Payload() Payload

	// Error return the last error observed by the source.
	Error() error
}

type Sink interface {
	// Consume process a Payload instance that's outputted by the pipeline.
	Consume(context.Context, Payload) error
}

type Pipeline struct {
	stages []StageRunner
}

// New return a new Pipeline instance where input payloads will traverse
// each one of the stages
func New(stages ...StageRunner) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

// Process reads the contents of the specified source, sends them through the
// various stages of the pipeline and directs the results to the specified sink
// and returns back any errors that may have occurred.
//
// Calls to Process block until:
//  - all data from the source has been processed OR
//  - an error occurs OR
//  - the supplied context expires/cancelled
//
// It is safe to call Process concurrently with different sources and sinks.
func (p *Pipeline) Process(ctx context.Context, source Source, sink Sink) error {

	var wg *sync.WaitGroup
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Allocates channels for wiring together the source, the pipeline stages
	// and the output sink. The output of the ith stage is used as an input
	// for the i+1 th stage. We need to allocate one extra channel than the
	// number of stages so we can also wire the source/sink.
	stageCh := make([]chan Payload, len(p.stages)+1)
	errCh := make(chan error, len(p.stages)+2)
	for i := range stageCh {
		stageCh[i] = make(chan Payload)
	}

	// start each stage worker
	wg.Add(len(p.stages))
	for i := range p.stages {
		go func(StageIdx int) {
			defer wg.Done()
			p.stages[StageIdx].Run(
				ctx,
				&WorkerParams{
					Stage: StageIdx,
					InCh:  stageCh[StageIdx],
					OutCh: stageCh[StageIdx+1],
					ErrCh: errCh,
				},
			)
			close(stageCh[StageIdx+1])
		}(i)
	}

	// start source and sink goruntines
	wg.Add(2)
	go func() {
		defer wg.Done()
		sourceWorker(ctx, source, stageCh[0], errCh)
		close(stageCh[0])
	}()

	go func() {
		defer wg.Done()
		sinkWorker(ctx, sink, stageCh[len(stageCh)-1], errCh)
	}()

	// garding goruntine closes the error channel and canceling the context
	// once all work is done.
	go func() {
		wg.Wait()
		close(errCh)
		ctxCancel()
	}()

	// collect errors
	var err error
	for pErr := range errCh {
		err = multierror.Append(err, pErr)
		ctxCancel()
	}
	return err
}
