/*
   implemeted the BSP https://en.wikipedia.org/wiki/Bulk_synchronous_parallel computing model to aid in
   processing graph data
*/
package bsp

import "github.com/Ahmed-Sermani/go-crawler/bsp/message"

type Vertex[VT any, ET any] struct {
	id     string
	value  VT
	active bool
	// two queues are needed
	// one queue to hold the messages for the current super-step
	// and another queue to buffer the messages for the next super-step
	// The queue at index super-step%2 hold the messages for the current super-step
	// the queue at (super-step + 1)%2 buffer the messages for the next super-step
	msgQueue [2]*message.Queue
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
