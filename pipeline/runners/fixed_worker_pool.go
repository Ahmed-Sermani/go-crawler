package runners

import (
	"context"
	"sync"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
)

type fixedWorkerPool struct {
	runners []pipeline.StageRunner
}

// StaticWorkerPool returns a StageRunner that starts a pool of numWorkers to process
// incoming payload in parallel.
func FixedWorkerPool(proc pipeline.Processor, numWorkers int) pipeline.StageRunner {
	if numWorkers <= 0 {
		panic("number of workers should be greater than 0")
	}
	runners := make([]pipeline.StageRunner, numWorkers)
	for i := range runners {
		runners[i] = FIFO(proc)
	}

	return &fixedWorkerPool{runners: runners}
}

func (runner *fixedWorkerPool) Run(ctx context.Context, params pipeline.StageParams) {
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
