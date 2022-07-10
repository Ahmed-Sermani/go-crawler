package pipeline

import (
	"context"

	"golang.org/x/xerrors"
)

// sourceWorker is a worker that consumes payloads from a Source and
// pushs them into an output channel that is used as an input for the first
// stage of the pipeline.

func sourceWorker(ctx context.Context, source Source, outCh chan<- Payload, errCh chan<- error) {
	for source.Next(ctx) {
		payload := source.Payload()
		select {
		case outCh <- payload:
		case <-ctx.Done():
			return
		}
	}

	if err := source.Error(); err != nil {
		wErr := xerrors.Errorf("sourceWorker: %w", err)
		select {
		case errCh <- wErr:
		default: // error channel is full.
		}
	}

}

// sinkWorker is a worker that consumes payloads from a provided input channel
// that passes them the provided sink (the input channel is the output channel of the last stage in the pipeline)
// it marks the outputed payloads as processed
func sinkWorker(ctx context.Context, sink Sink, inCh <-chan Payload, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload, open := <-inCh:
			if !open {
				return
			}
			if err := sink.Consume(ctx, payload); err != nil {
				wErr := xerrors.Errorf("sinkWorker: %w", err)
				select {
				case errCh <- wErr:
				default: // error channel is full.
				}
			}
			payload.MarkAsProcessed()
		}
	}
}
