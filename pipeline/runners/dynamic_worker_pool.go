package runners

import (
	"context"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
	"golang.org/x/xerrors"
)

type dynamicWorkerPool struct {
	proc pipeline.Processor
	// using as a concurrency control mechanism to implemet the dynamci pool
	tPool chan struct{}
}

// DynamicWorkerPool returns a StageRunner that implemets a dynamic worker pool
// that can scale up to maxWorkers for processing incomming payloads concurrently
func DynamicWorkerPool(proc pipeline.Processor, maxWorkers int) pipeline.StageRunner {
	if maxWorkers <= 0 {
		panic("DynamicWorkerPool: maxWorkers can be negative or equal to zero")
	}
	tPool := make(chan struct{}, maxWorkers)
	for i := 0; i <= maxWorkers; i++ {
		tPool <- struct{}{}
	}
	return &dynamicWorkerPool{proc: proc, tPool: tPool}
}

func (p *dynamicWorkerPool) Run(ctx context.Context, params pipeline.StageParams) {
	for {
		select {
		case <-ctx.Done():
			break
		case payloadIn, open := <-params.Input():

			if !open {
				break
			}

			// optain a token
			// blocks until it gets one or the context is cancelled
			var t struct{}
			select {
			case t = <-p.tPool:
			case <-ctx.Done():
				break
			}

			// run the work in a goruntine
			// when it ends it retruns the token to tPool
			go func(payloadIn pipeline.Payload, t struct{}) {
				defer func() { p.tPool <- t }()
				payloadOut, err := p.proc.Process(ctx, payloadIn)
				if err != nil {
					wErr := xerrors.Errorf("pipeline stage %d: %w", params.StageIndex(), err)
					emitError(wErr, params.Error())
					return
				}

				if payloadOut == nil {
					payloadIn.MarkAsProcessed()
					return
				}

				// output processed data
				select {
				case params.Output() <- payloadOut:
				case <-ctx.Done():
				}

			}(payloadIn, t)

		}
	}
	// make sure all work are exit before returning
	// emptying tPool
	for i := 0; i < cap(p.tPool); i++ {
		<-p.tPool
	}
}
