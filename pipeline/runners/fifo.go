package runners

import (
	"context"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
	"golang.org/x/xerrors"
)

type fifo struct {
	proc pipeline.Processor
}

// FIFO returns a StageRunner that process payloads in first-in-first-out fashion.
// Input passed to the specified processor and output to the next stage.
func FIFO(proc pipeline.Processor) pipeline.StageRunner {
	return fifo{proc: proc}
}

func (runner fifo) Run(ctx context.Context, params pipeline.StageParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload, open := <-params.Input():
			if !open {
				return
			}
			processedPayload, err := runner.proc.Process(ctx, payload)
			if err != nil {
				emitError(
					xerrors.Errorf("pipeline stage %d: %w", params.StageIndex(), err),
					params.Error(),
				)
			}
			// if the processor does not output a payload.
			// then discard since there's nothing to do in the next stage.
			if processedPayload == nil {
				payload.MarkAsProcessed()
				continue
			}

			// send the processedPayload to the next stage.
			select {
			case params.Output() <- processedPayload:
			case <-ctx.Done():
				return
			}
		}
	}
}
