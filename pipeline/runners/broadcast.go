package runners

import (
	"context"
	"sync"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
)

type broadcast struct {
	fifos []pipeline.StageRunner
}

func Broadcast(procs ...pipeline.Processor) pipeline.StageRunner {
	if len(procs) <= 0 {
		panic("Broadcast: at least one processor must be specified")
	}

	fifos := make([]pipeline.StageRunner, len(procs))
	for i, p := range procs {
		fifos[i] = FIFO(p)
	}

	return &broadcast{fifos: fifos}
}

func (b *broadcast) Run(ctx context.Context, params pipeline.StageParams) {
	var (
		// for concurrency control
		wg *sync.WaitGroup
		// broadcastor channels
		pInCh = make([]chan pipeline.Payload, len(b.fifos))
	)

	// start each fifo in a go-runtine, each fifo gets its own non-buffered input channels.
	// output and error channels say shared
	for i := range b.fifos {
		wg.Add(1)
		pInCh[i] = make(chan pipeline.Payload)
		go func(idx int) {
			defer wg.Done()
			params := &pipeline.WorkerParams{
				Stage: params.StageIndex(),
				InCh:  pInCh[idx],
				OutCh: params.Output(),
				ErrCh: params.Error(),
			}
			b.fifos[idx].Run(ctx, params)
		}(i)
	}
done:
	for {
		select {
		case <-ctx.Done():
			break
		case payloadIn, open := <-params.Input():
			if !open {
				break
			}

			// breadcast the payload
			for i := len(b.fifos) - 1; i >= 0; i-- {
				// clone the payload for each fifo to avoid
				// data races except the first one
				fifoPayload := payloadIn
				if i != 0 {
					fifoPayload = fifoPayload.Clone()
				}

				select {
				case <-ctx.Done():
					break done
				case pInCh[i] <- fifoPayload:
					// sent to ith fifo
				}
			}
		}
	}
	// close the breadcasting channel and wait for all fifos to exit
	for _, ch := range pInCh {
		close(ch)
	}
	wg.Wait()
}
