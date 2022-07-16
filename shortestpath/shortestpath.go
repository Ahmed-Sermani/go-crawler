/*
	implements a parallel Dijkstra for find shortest path from a vertex to any other vertex
	in the graph.
*/
package shortestpath

import (
	"context"
	"math"

	"github.com/Ahmed-Sermani/go-crawler/bsp"
	"github.com/Ahmed-Sermani/go-crawler/bsp/message"
	"golang.org/x/xerrors"
)

// PathCostMessage is used to advertise the cost of a path through a vertex.
type PathCostMessage struct {
	// The ID of the vertex this cost announcement originates from.
	FromID string

	// The cost of the path from this vertex to the source vertex via FromID.
	Cost int
}

func (pc PathCostMessage) Type() string { return "cost" }

type pathState struct {
	minDist    int
	prevInPath string
}

// Calculator implements a shortest path calculator from a single vertex to
// all other vertices in a connected graph.
type Calculator struct {
	g     *bsp.Graph[*pathState, int]
	srcID string

	executorFactory bsp.ExecutorFactory[*pathState, int]
}

// NewCalculator returns a new shortest path calculator instance.
func NewCalculator(numWorkers int) (*Calculator, error) {
	c := &Calculator{
		executorFactory: bsp.NewExecutor[*pathState, int],
	}

	var err error
	if c.g, err = bsp.NewGraph(bsp.GraphConfig[*pathState, int]{
		ComputeFn:      c.findShortestPath,
		ComputeWorkers: numWorkers,
	}); err != nil {
		return nil, err
	}

	return c, nil
}

// Close cleans up any allocated graph resources.
func (c *Calculator) Close() error {
	return c.g.Close()
}

// SetExecutorFactory configures the calculator to use the a custom executor
// factory when CalculateShortestPaths is invoked.
func (c *Calculator) SetExecutorFactory(factory bsp.ExecutorFactory[*pathState, int]) {
	c.executorFactory = factory
}

// AddVertex inserts a new vertex with the specified ID into the graph.
func (c *Calculator) AddVertex(id string) {
	c.g.AddVertex(id, nil)
}

// AddEdge creates a directed edge from srcID to dstID with the specified cost.
// An error will be returned if a negative cost value is specified.
func (c *Calculator) AddEdge(srcID, dstID string, cost int) error {
	if cost < 0 {
		return xerrors.Errorf("negative edge costs not supported")
	}
	return c.g.AddEdge(srcID, dstID, cost)
}

// CalculateShortestPaths finds the shortest path costs from srcID to all other
// vertices in the graph.
func (c *Calculator) CalculateShortestPaths(ctx context.Context, srcID string) error {
	c.srcID = srcID
	exec := c.executorFactory(c.g, bsp.ExecutorHooks[*pathState, int]{
		PostStepKeepRunning: func(_ context.Context, _ *bsp.Graph[*pathState, int], activeInStep int) (bool, error) {
			return activeInStep != 0, nil
		},
	})
	return exec.RunToCompletion(ctx)
}

// ShortestPathTo returns the shortest path from the source vertex to the
// specified destination together with its cost.
func (c *Calculator) ShortestPathTo(dstID string) ([]string, int, error) {
	vertMap := c.g.Vertices()
	v, exists := vertMap[dstID]
	if !exists {
		return nil, 0, xerrors.Errorf("unknown vertex with ID %q", dstID)
	}

	var (
		minDist = v.Value().minDist
		path    []string
	)

	for ; v.ID() != c.srcID; v = vertMap[v.Value().prevInPath] {
		path = append(path, v.ID())
	}
	path = append(path, c.srcID)

	// Reverse in place to get path from src->dst
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path, minDist, nil
}

func (c *Calculator) findShortestPath(
	g *bsp.Graph[*pathState, int],
	v *bsp.Vertex[*pathState, int],
	msgIt message.Iterator,
) error {
	if g.Superstep() == 0 {
		v.SetValue(&pathState{
			minDist: int(math.MaxInt64),
		})
	}

	minDist := int(math.MaxInt64)
	if v.ID() == c.srcID {
		minDist = 0
	}

	// Process cost messages from neighbors and update minDist if
	// we receive a better path announcement.
	var via string
	for msgIt.Next() {
		m := msgIt.Message().(*PathCostMessage)
		if m.Cost < minDist {
			minDist = m.Cost
			via = m.FromID
		}
	}

	// If a better path was found through this vertex, announce it
	// to all neighbors so they can update their own scores.
	st := v.Value()
	if minDist < st.minDist {
		st.minDist = minDist
		st.prevInPath = via
		for _, e := range v.Edges() {
			costMsg := &PathCostMessage{
				FromID: v.ID(),
				Cost:   minDist + e.Value(),
			}
			if err := g.SendMessage(e.DstID(), costMsg); err != nil {
				return err
			}
		}
	}

	// We are done unless we receive a better path announcement.
	v.Freeze()
	return nil
}
