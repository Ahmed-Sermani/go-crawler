package ranker

import (
	"math"

	"github.com/Ahmed-Sermani/go-crawler/bsp"
	"github.com/Ahmed-Sermani/go-crawler/bsp/message"
)

// IncomingScoreMessage is used for distributing PageRank scores to neighbors.
type IncomingScoreMessage struct {
	Score float64
}

func (pr IncomingScoreMessage) Type() string { return "score" }

// makeRankerComputeFunc returns a ComputeFunc that executes the PageRank ranker 
// algorithm using the provided dampingFactor value.
func makeRankerComputeFunc(dampingFactor float64) bsp.ComputeFunc[float64, any] {
	return func(g *bsp.Graph[float64, any], v *bsp.Vertex[float64, any], msgIt message.Iterator) error {
		superstep := g.Superstep()
		pageCountAgg := g.Aggregator("page_count")

		// At step 0, we use an aggregator to count the number of vertices in the graph.
		if superstep == 0 {
			pageCountAgg.Aggregate(1)
			return nil
		}

		var (
			pageCount = float64(pageCountAgg.Get().(int))
			newScore  float64
		)
		switch superstep {
		case 1:
			// At step 1 evenly distribute the PageRank scores across all
			// vertices. As the sum of all scores should be equal to 1, each vertex
			// is assigned an initial score of 1/pageCount.
			newScore = 1.0 / pageCount
		default:
			// Process incoming messages and calculate new score.
			newScore = (1.0 - dampingFactor) / pageCount
			for msgIt.Next() {
				score := msgIt.Message().(IncomingScoreMessage).Score
				newScore += dampingFactor * score
			}

			// Add accumulated residual page rank from any dead-ends
			// encountered during the previous step.
			resAggr := g.Aggregator(residualInputAccName(superstep))
			newScore += dampingFactor * resAggr.Get().(float64)
		}

		absDelta := math.Abs(v.Value() - newScore)
		g.Aggregator("SAD").Aggregate(absDelta)

		v.SetValue(newScore)

		// If this is a dead-end (no outgoing links) we treat this link
		// as if it was being connected to all links in the graph.
		// Since we cannot broadcast a message to all vertices we will
		// add the per-vertex residual score to an accumulator and
		// integrate it into the scores calculated over the next round.
		numOutLinks := float64(len(v.Edges()))
		if numOutLinks == 0.0 {
			g.Aggregator(residualOutputAccName(superstep)).Aggregate(newScore / pageCount)
			return nil
		}

		// Otherwise, evenly distribute this node's score to all its
		// neighbors.
		return g.BroadcastToNeighbors(v, IncomingScoreMessage{newScore / numOutLinks})
	}
}

