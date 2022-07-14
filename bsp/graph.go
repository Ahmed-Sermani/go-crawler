package bsp

import (
	"github.com/Ahmed-Sermani/go-crawler/bsp/message"
	"golang.org/x/xerrors"
)

// ComputeFunc is a function that a graph instance invokes on each vertex when
// executing a superstep.
type ComputeFunc[VT, ET any] func(g *Graph[VT, ET], v *Vertex[VT, ET], msgIt message.Iterator) error

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
	aggregators  map[string]Aggregator
	relayer      Relayer
	computeFunc  ComputeFunc[VT, ET]
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

func (g *Graph[VT, ET]) RegisterAggregator(name string, aggregator Aggregator) {
	g.aggregators[name] = aggregator
}

// RegisterRelayer configures a Relayer that the graph will invoke when
// attempting to deliver a message to a vertex that is not known locally but
// could potentially be owned by a remote graph instance.
func (g *Graph[VT, ET]) RegisterRelayer(relayer Relayer) { g.relayer = relayer }

func (g *Graph[VT, ET]) Aggregator(name string) Aggregator {
	return g.aggregators[name]
}

func (g *Graph[VT, ET]) Aggregators() map[string]Aggregator { return g.aggregators }

// BroadcastToNeighbors broadcasts the provided message to the neighboring vertecies (local or remote).
// Neighbors will recive the message in the next super-step
func (g *Graph[VT, ET]) BroadcastToNeighbors(v *Vertex[VT, ET], msg message.Message) error {
	for _, e := range v.edges {
		if err := g.SendMessage(e.DstID(), msg); err != nil {
			return err
		}
	}
	return nil
}

// SendMessage attempts to deliver a message to the vertex with the specified
// destination ID. Messages are queued for delivery and will be processed by
// receipients in the next superstep.
//
// If the destination ID is not known by this graph, it might still be a valid
// ID for a vertex that is owned by a remote graph instance. If the client has
// provided a Relayer when configuring the graph, SendMessage will delegate
// message delivery to it.
//
// On the other hand, if no Relayer is defined or the configured
// RemoteMessageSender returns a ErrDestinationIsLocal error, SendMessage will
// first check whether an UnknownVertexHandler has been provided at
// configuration time and invoke it. Otherwise, an ErrInvalidMessageDestination
// is returned to the caller.
func (g *Graph[VT, ET]) SendMessage(dst string, msg message.Message) error {
	dstVertex := g.vertices[dst]
	if dstVertex != nil {
		// send the message locally to reciver vertex in the next superstep
		return dstVertex.msgQueue[(g.superstep+1)%2].Enqueue(msg)
	}

	// the vertex is not know locally. This could mean that the vertex is partitioned to be
	// processed by a remote graph instance running on a another node.
	// the relayer is defined. Delegate message passing to it.
	if g.relayer != nil {
		if err := g.relayer.Relay(dst, msg); !xerrors.Is(err, ErrDestinationIsLocal) {
			return err
		}
	}

	return xerrors.Errorf("can't deliver message to %q: %w", dst, ErrInvalidMessageDestination)
}
