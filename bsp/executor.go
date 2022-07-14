package bsp

import "context"

// Executor wraps a Graph instance and provides an orchestration layer for
// executing supersteps until an error occurs or an exit condition is met.
// Users can provide an optional set of hooks to be executed before and
// after each super-step.
type Executor[VT, ET any] struct {
	g  *Graph[VT, ET]
	cb ExecutorHooks[VT, ET]
}

// NewExecutor returns an Executor instance for graph g that invokes the
// provided list of hooks inside each execution loop.
func NewExecutor[VT, ET any](g *Graph[VT, ET], cb ExecutorHooks[VT, ET]) *Executor[VT, ET] {
	if cb.PreStep == nil {
		cb.PreStep = func(context.Context, *Graph[VT, ET]) error { return nil }
	}
	if cb.PostStep == nil {
		cb.PostStep = func(context.Context, *Graph[VT, ET], int) error { return nil }
	}
	if cb.PostStepKeepRunning == nil {
		cb.PostStepKeepRunning = func(context.Context, *Graph[VT, ET], int) (bool, error) { return true, nil }
	}
	g.superstep = 0
	return &Executor[VT, ET]{
		g:  g,
		cb: cb,
	}
}

// ExecutorHooks encapsulates a series of hooks that are invoked by an
// Executor instance on a graph. All hooks are optional and will be ignored
// if not specified.
type ExecutorHooks[VT, ET any] struct {
	// PreStep, if defined, is invoked before running the next superstep.
	// This is a good place to initialize variables, aggregators etc. that
	// will be used for the next superstep.
	PreStep func(ctx context.Context, g *Graph[VT, ET]) error

	// PostStep, if defined, is invoked after running a superstep.
	PostStep func(ctx context.Context, g *Graph[VT, ET], activeInStep int) error

	// PostStepKeepRunning, if defined, is invoked after running a superstep
	// to decide whether the stop condition for terminating the run has
	// been met. The number of the active vertices in the last step is
	// passed as the second argument.
	PostStepKeepRunning func(ctx context.Context, g *Graph[VT, ET], activeInStep int) (bool, error)
}

// RunToCompletion keeps executing supersteps until the context expires, an
// error occurs or one of the Pre/PostStepKeepRunning callbacks specified at
// configuration time returns false.
func (ex *Executor[VT, ET]) RunToCompletion(ctx context.Context) error {
	return ex.run(ctx, -1)
}

// RunSteps executes at most numStep supersteps unless the context expires, an
// error occurs or one of the Pre/PostStepKeepRunning callbacks specified at
// configuration time returns false.
func (ex *Executor[VT, ET]) RunSteps(ctx context.Context, numSteps int) error {
	return ex.run(ctx, numSteps)
}

// Graph returns the graph instance associated with this executor.
func (ex *Executor[VT, ET]) Graph() *Graph[VT, ET] {
	return ex.g
}

// Superstep returns the current graph superstep.
func (ex *Executor[VT, ET]) Superstep() int {
	return ex.g.Superstep()
}


func (ex *Executor[VT, ET]) run(ctx context.Context, maxSteps int) error {
	var (
		activeInStep int
		err          error
		keepRunning  bool
		cb           = ex.cb
	)

	for ; maxSteps != 0; ex.g.superstep, maxSteps = ex.g.superstep+1, maxSteps-1 {
		// check for context cancel before the start of each step
		select {
		case <-ctx.Done():
			break
		default:
		}
		
		// runs hooks and the superstep
		if err = cb.PreStep(ctx, ex.g); err != nil {
			break
		} else if activeInStep, err = ex.g.step(); err != nil {
			break
		} else if err = cb.PostStep(ctx, ex.g, activeInStep); err != nil {
			break
		} else if keepRunning, err = cb.PostStepKeepRunning(ctx, ex.g, activeInStep); !keepRunning || err != nil {
			break
		}
	}
	return err
}
