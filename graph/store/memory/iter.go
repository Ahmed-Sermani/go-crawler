package memory

import "github.com/Ahmed-Sermani/go-crawler/graph"

type linkIterator struct {
	s *InMemoryGraph

	links  []*graph.Link
	curIdx int
}

func (i *linkIterator) Next() bool {
	if i.curIdx >= len(i.links) {
		return false
	}
	i.curIdx++
	return true
}

func (i *linkIterator) Link() *graph.Link {
	i.s.mu.RLock()
	link := new(graph.Link)
	*link = *i.links[i.curIdx-1]
	i.s.mu.RUnlock()
	return link
}

func (i *linkIterator) Error() error {
	return nil
}

func (i *linkIterator) Close() error {
	return nil
}

// edgeIterator is a graph.EdgeIterator implementation for the in-memory graph.
type edgeIterator struct {
	s *InMemoryGraph

	edges    []*graph.Edge
	curIndex int
}

// Next implements graph.LinkIterator.
func (i *edgeIterator) Next() bool {
	if i.curIndex >= len(i.edges) {
		return false
	}
	i.curIndex++
	return true
}

func (i *edgeIterator) Error() error {
	return nil
}

func (i *edgeIterator) Close() error {
	return nil
}

func (i *edgeIterator) Edge() *graph.Edge {
	// The edge pointer contents may be overwritten by a graph update; to
	// avoid data-races we acquire the read lock first and clone the edge
	i.s.mu.RLock()
	edge := new(graph.Edge)
	*edge = *i.edges[i.curIndex-1]
	i.s.mu.RUnlock()
	return edge
}
