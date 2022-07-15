/*
   Implemets Google famous and first
   PageRank algorithm https://en.wikipedia.org/wiki/PageRank
*/
package ranker

import (
	"context"

	"github.com/Ahmed-Sermani/go-crawler/bsp"
	"github.com/Ahmed-Sermani/go-crawler/bsp/aggregators"
	"golang.org/x/xerrors"
)

/*
   PageRank works by counting the number and quality of links to
   a page to determine a rough estimate of how important the website is.
   The underlying assumption is that more important websites are likely
   to receive more links from other websites.

   To calculate the score for each vertex in the graph,
   the PageRank algorithm utilizes the model of the random surfer.
   Under this model, a surfer performs an initial search and lands on a page from the graph.
   From that point on, surfers randomly select one of the following two options:

       They can follow any outgoing link from the current page and navigate to a new page.
       Surfers choose this option with a predefined probability that we will be referring to with the term damping factor.

       Alternatively, they can decide to run a new search query.
       This decision has the effect of teleporting the surfer to a random page in the graph.

   The PageRank algorithm works under the assumption that the preceding steps are repeated in perpetuity.
   As a result, the model is equivalent to performing a random walk of the page graph.
   PageRank score values reflect the probability that a surfer lands on a particular page.
   By this definition, we expect the following to occur
       Each PageRank score should be a value in the [0, 1] range
       The sum of all assigned PageRank scores should be exactly equal to 1
*/

// Ranker executes the iterative version of the PageRank algorithm
// on a graph until the desired level of convergence is reached.
type Ranker struct {
	g   *bsp.Graph[float64, any]
	cfg Config

	executorFactory bsp.ExecutorFactory[float64, any]
}

// NewRanker returns a new Ranker instance using the provided config
// options.
func NewRanker(cfg Config) (*Ranker, error) {
	if err := cfg.validate(); err != nil {
		return nil, xerrors.Errorf("PageRank ranker config validation failed: %w", err)
	}

	g, err := bsp.NewGraph(bsp.GraphConfig[float64, any]{
		ComputeWorkers: cfg.ComputeWorkers,
		ComputeFn:      makeRankerComputeFunc(cfg.DampingFactor),
	})
	if err != nil {
		return nil, err
	}

	return &Ranker{
		cfg:             cfg,
		g:               g,
		executorFactory: bsp.NewExecutor[float64, any],
	}, nil
}

// Close releases any resources allocated by this PageRank ranker instance.
func (c *Ranker) Close() error {
	return c.g.Close()
}

// SetExecutorFactory configures the ranker to use the a custom executor
// factory when the Executor method is invoked.
func (c *Ranker) SetExecutorFactory(factory bsp.ExecutorFactory[float64, any]) {
	c.executorFactory = factory
}

// AddVertex inserts a new vertex to the graph with the given id.
func (c *Ranker) AddVertex(id string) {
	c.g.AddVertex(id, 0.0)
}

// AddEdge inserts a directed edge from src to dst. If both src and dst refer
// to the same vertex then this is a no-op.
func (c *Ranker) AddEdge(src, dst string) error {
	// Don't allow self-links
	if src == dst {
		return nil
	}
	return c.g.AddEdge(src, dst, nil)
}

// Graph returns the underlying bspgraph.Graph instance.
func (c *Ranker) Graph() *bsp.Graph[float64, any] {
	return c.g
}

// Executor creates and return a bspgraph.Executor for running the PageRank
// algorithm once the graph layout has been properly set up.
func (c *Ranker) Executor() *bsp.Executor[float64, any] {
	c.registerAggregators()
	cb := bsp.ExecutorHooks[float64, any]{
		PreStep: func(_ context.Context, g *bsp.Graph[float64, any]) error {
			// Reset sum of abs differences aggregator and residual
			// aggregator for next step.
			g.Aggregator("SAD").Set(0.0)
			g.Aggregator(residualOutputAccName(g.Superstep())).Set(0.0)
			return nil
		},
		PostStepKeepRunning: func(_ context.Context, g *bsp.Graph[float64, any], _ int) (bool, error) {
			// Supersteps 0 and 1 are part of the algorithm initialization;
			// the predicate should only be evaluated for supersteps > 1
			sad := c.g.Aggregator("SAD").Get().(float64)
			return !(g.Superstep() > 1 && sad < c.cfg.MinSADForConvergence), nil
		},
	}

	return c.executorFactory(c.g, cb)
}

// registerAggregators creates and registers the aggregator instances that we
// need to run the PageRank ranker algorithm.
func (c *Ranker) registerAggregators() {
	c.g.RegisterAggregator("page_count", new(aggregators.IntAggregator))
	c.g.RegisterAggregator("residual_0", new(aggregators.Float64Aggregator))
	c.g.RegisterAggregator("residual_1", new(aggregators.Float64Aggregator))
	c.g.RegisterAggregator("SAD", new(aggregators.Float64Aggregator))
}

// Scores invokes the provided visitor function for each vertex in the graph.
func (c *Ranker) Scores(visitFn func(id string, score float64) error) error {
	for id, v := range c.g.Vertices() {
		if err := visitFn(id, v.Value()); err != nil {
			return err
		}
	}
	return nil
}

// residualOutputAccName returns the name of the aggregator where the
// residual PageRank scores for the specified superstep are to be written to.
func residualOutputAccName(superstep int) string {
	if superstep%2 == 0 {
		return "residual_0"
	}
	return "residual_1"
}

// residualInputAccName returns the name of the aggregator where the
// residual PageRank scores for the specified superstep are to be read from.
func residualInputAccName(superstep int) string {
	if (superstep+1)%2 == 0 {
		return "residual_0"
	}
	return "residual_1"
}


