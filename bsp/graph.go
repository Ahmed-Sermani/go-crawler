package bsp

import (
	"sync"
	"sync/atomic"

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

	// wg used for compute workers
	wg sync.WaitGroup

	// vertexCh polled by compute function workers to optain the next
	// to be processed
	vertexCh chan *Vertex[VT, ET]

	// errCh is buffered channel where workers publish any errors that occurs
	// during invokion the compute function.
	// When compute worker detects an error it will attempts to publish it into the channel.
	// if the channel is full, another error has already been written to it, the new error will
	// be safely ignored.
	errCh chan error

	// stepCompletedCh channel allows compute workers to signal
	// when the last enqueued vertex has been processed.
	stepCompletedCh chan struct{}

	// activeInStep is the number of vertices that are processed in the superstep
	// it will be reset at the start of superstep
	activeInStep int64

	// pendingInStep is the number of pending vertices to be processed in the superstep
	// it's set to len(vertices) at the start of the superstep
	pendingInStep int64
}

// NewGraph creates a new Graph instance using the specified configuration. It
// is important for callers to invoke Close() on the returned graph instance
// when they are done using it.
func NewGraph[VT, ET any](cfg GraphConfig[VT, ET]) (*Graph[VT, ET], error) {
	if err := cfg.validate(); err != nil {
		return nil, xerrors.Errorf("graph config validation failed: %w", err)
	}

	g := &Graph[VT, ET]{
		computeFunc:  cfg.ComputeFn,
		queueFactory: cfg.QueueFactory,
		aggregators:  make(map[string]Aggregator),
		vertices:     make(map[string]*Vertex[VT, ET]),
	}
	g.startWorkers(cfg.ComputeWorkers)

	return g, nil
}

// Close releases any resources associated with the graph.
func (g *Graph[VT, ET]) Close() error {
	close(g.vertexCh)
	g.wg.Wait()

	return g.Reset()
}

// Reset the state of the graph by removing any existing vertices or
// aggregators and resetting the superstep counter.
func (g *Graph[VT, ET]) Reset() error {
	g.superstep = 0
	for _, v := range g.vertices {
		for i := 0; i < 2; i++ {
			if err := v.msgQueue[i].Close(); err != nil {
				return xerrors.Errorf("closing message queue #%d for vertex %v: %w", i, v.ID(), err)
			}
		}
	}
	g.vertices = make(map[string]*Vertex[VT, ET])
	g.aggregators = make(map[string]Aggregator)
	return nil
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

func (g *Graph[VT, ET]) Superstep() int { return g.superstep }

func (g *Graph[VT, ET]) Vertices() map[string]*Vertex[VT, ET] { return g.vertices }

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

// startWorkers allocates the required channels and spins up numWorkers to
// execute each superstep.
func (g *Graph[VT, ET]) startWorkers(numWorkers int) {
	g.vertexCh = make(chan *Vertex[VT, ET])
	g.errCh = make(chan error, 1)
	g.stepCompletedCh = make(chan struct{})

	g.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go g.stepWorker()
	}
}

// stepWorker consumes vertexCh for incoming vertices and executes the configured
// ComputeFunc for each one. The worker exits when vertexCh gets
// closed.
func (g *Graph[VT, ET]) stepWorker() {
	defer g.wg.Done()
	for v := range g.vertexCh {
		stepMsgQueueBuffer := g.superstep % 2
		if v.active || v.msgQueue[stepMsgQueueBuffer].PendingMessages() {
			_ = atomic.AddInt64(&g.activeInStep, 1)
			v.active = true

			// execute the compute funciton on the vertex
			err := g.computeFunc(g, v, v.msgQueue[stepMsgQueueBuffer].Messages())
			if err != nil {
				emitError(g.errCh, xerrors.Errorf("error while running compute function for vertex %q: %w", v.ID(), err))
				// flush non-comsumed messages
			} else if err := v.msgQueue[stepMsgQueueBuffer].DiscardMessages(); err != nil {
				emitError(g.errCh, xerrors.Errorf("failed discarding un-processed message for vertex %q: %w", v.ID(), err))
			}
		}
		if atomic.AddInt64(&g.pendingInStep, -1) == 0 {
			g.stepCompletedCh <- struct{}{}
		}
	}
}

// Step executes the next superstep and returns back the number of vertices
// that were processed either because they were still active or because they
// received a message.
func (g *Graph[VT, ET]) step() (int, error) {
	// at the start of the superstep
	// it's safe to assgin values to these variables directly
	g.activeInStep = 0
	g.pendingInStep = int64(len(g.vertices))

	// no work to do
	if g.pendingInStep == 0 {
		return 0, nil
	}

	// send vertices to the channel to be processed
	for _, v := range g.vertices {
		g.vertexCh <- v
	}

	// block until the worker pool has finished processing all vertices
	<-g.stepCompletedCh

	// get errors happend during the executing the step
	var err error
	select {
	case err = <-g.errCh:
	default: // no error
	}

	return int(g.activeInStep), err
}

func emitError(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default: // the channel already contains an error
	}
}
