/*
   implemeted the BSP https://en.wikipedia.org/wiki/Bulk_synchronous_parallel computing model to aid in
   processing graph data
*/
package bsp

import (
	"github.com/Ahmed-Sermani/go-crawler/bsp/message"
	"golang.org/x/xerrors"
)

var ErrUnknownEdgeSource = xerrors.New("source vertex is not part of the graph")

type Vertex[VT any, ET any] struct {
	id     string
	value  VT
	active bool
	// two queues are needed
	// one queue to hold the messages for the current super-step
	// and another queue to buffer the messages for the next super-step
	// The queue at index super-step%2 hold the messages for the current super-step
	// the queue at (super-step + 1)%2 buffer the messages for the next super-step
	msgQueue [2]message.Queue
	edges    []*Edge[ET]
}

func (v *Vertex[VT, ET]) ID() string { return v.id }

func (v *Vertex[VT, ET]) Edges() []*Edge[ET] { return v.edges }

// Freeze marks the vertex as inactive. Inactive vertices will not be processed
// in the following supersteps unless they receive a message in which case they
// will be re-activated.
func (v *Vertex[VT, ET]) Freeze() { v.active = false }

func (v *Vertex[VT, ET]) Value() VT { return v.value }

func (v *Vertex[VT, ET]) SetValue(val VT) { v.value = val }

type Edge[ET any] struct {
	value ET
	dstID string
}

func (e *Edge[ET]) DstID() string { return e.dstID }

func (e *Edge[ET]) Value() ET { return e.value }

func (e *Edge[ET]) SetValue(val ET) { e.value = val }

// Graph implements a parallel graph processor based on the concepts described
// in the Pregel paper https://15799.courses.cs.cmu.edu/fall2013/static/papers/p135-malewicz.pdf .
type Graph[VT, ET any] struct {
	superstep    int
	vertices     map[string]*Vertex[VT, ET]
	queueFactory message.QueueFactory
}

// AddVertex inserts a new vertex with the specified id and initial value into
// the graph. If the vertex already exists, AddVertex will just overwrite its
// value with the provided initValue.
func (g *Graph[VT, ET]) AddVertex(id string, initValue VT) {
	v := g.vertices[id]
	if v == nil {
		v = &Vertex[VT, ET]{
			id: id,
			msgQueue: [2]message.Queue{
				g.queueFactory(),
				g.queueFactory(),
			},
			active: true,
		}
		g.vertices[id] = v
	}
	v.SetValue(initValue)
}

// AddEdge inserts a directed edge from src to destination and annotates it
// with the specified initValue. By design, edges are owned by the source
// and therefore srcID must resolve to a local vertex. Otherwise, AddEdge returns an error.
func (g *Graph[VT, ET]) AddEdge(srcID, dstID string, initValue ET) error {
	srcVertex := g.vertices[srcID]
	if srcVertex == nil {
		return xerrors.Errorf("create edge from %q to %q: %w", srcID, dstID, ErrUnknownEdgeSource)
	}

	srcVertex.edges = append(srcVertex.edges, &Edge[ET]{
		dstID: dstID,
		value: initValue,
	})
	return nil
}
