package bsp

import (
	"github.com/Ahmed-Sermani/go-crawler/bsp/message"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
)

// GraphConfig encapsulates the configuration options for creating graphs.
type GraphConfig[VT, ET any] struct {
	// QueueFactory is used by the graph to create message queue instances
	// for each vertex that is added to the graph. If not specified, the
	// default in-memory queue will be used instead.
	QueueFactory message.QueueFactory

	// ComputeFn is the compute function that will be invoked for each graph
	// vertex when executing a superstep. A valid ComputeFunc instance is
	// required for the config to be valid.
	ComputeFn ComputeFunc[VT, ET]

	// ComputeWorkers specifies the number of workers to use for invoking
	// the registered ComputeFunc when executing each superstep. If not
	// specified, a single worker will be used.
	ComputeWorkers int
}

func (g *GraphConfig[VT, ET]) validate() error {
	var err error
	if g.QueueFactory == nil {
		g.QueueFactory = message.NewInMemoryQueue
	}
	if g.ComputeWorkers <= 0 {
		g.ComputeWorkers = 1
	}

	if g.ComputeFn == nil {
		err = multierror.Append(err, xerrors.New("compute function not specified"))
	}

	return err
}
