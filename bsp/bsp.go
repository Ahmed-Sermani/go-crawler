/*
   implemeted the BSP https://en.wikipedia.org/wiki/Bulk_synchronous_parallel computing model to aid in
   processing graph data
*/
package bsp

import (
	"golang.org/x/xerrors"
)

var ErrUnknownEdgeSource = xerrors.New("source vertex is not part of the graph")

type Aggregator interface {
	Type() string
	Set(val any)
	Get() any
	// updates the Aggregator value based on the current value.
	Aggregate(val any)

	// Delta returns the change in the aggregator's value since the last
	// call to Delta. The delta values can be used in distributed
	// aggregator use-cases to reduce local, partially-aggregated values
	// into a single value across by feeding them into the Aggregate method
	// of a top-level aggregator.
	//
	// For example, in a distributed counter scenario, each node maintains
	// its own local counter instance. At the end of each step, the master
	// node calls delta on each local counter and aggregates the values
	// to obtain the correct total which is then pushed back to the workers.
	Delta() any
}
