package pipeline

import (
	"context"
	"sync"

	"golang.org/x/xerrors"
)

type fifo struct {
	proc Processor
}

// FIFO returns a StageRunner that process payloads in first-in-first-out fashion.
// Input passed to the specified processor and output to the next stage.
func FIFO(proc Processor) StageRunner {
	return fifo{proc: proc}
}

func (runner fifo) Run(ctx context.Context, params StageParams) {
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
			// if the processor dose not output a payload.
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

type staticWorkerPool struct {
	runners []StageRunner
}

// StaticWorkerPool returns a StageRunner that starts a pool of numWorkers to process
// incoming payload in parallel.
func StaticWorkerPool(proc Processor, numWorkers int) StageRunner {
	if numWorkers <= 0 {
		panic("number of workers should be greater than 0")
	}
	runners := make([]StageRunner, numWorkers)
	for i := range runners {
		runners[i] = FIFO(proc)
	}

	return &staticWorkerPool{runners: runners}
}

func (runner *staticWorkerPool) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup
	wg.Add(len(runner.runners))
	for i := range runner.runners {
		go func(runnerIndex int) {
			defer wg.Done()
			runner.runners[runnerIndex].Run(ctx, params)
		}(i)
	}
	wg.Wait()
}
