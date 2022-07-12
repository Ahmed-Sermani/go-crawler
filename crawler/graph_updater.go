package crawler

import (
	"time"

	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/Ahmed-Sermani/go-crawler/pipeline"
	"golang.org/x/net/context"
)

var _ pipeline.Processor = (*graphUpdater)(nil)

type graphUpdater struct {
	g Graph
}

func newGraphUpdater(g Graph) *graphUpdater {
	return &graphUpdater{
		g: g,
	}
}

func (gu *graphUpdater) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	src := &graph.Link{
		ID:          payload.LinkID,
		URL:         payload.URL,
		RetreivedAt: time.Now(),
	}
	if err := gu.g.UpsertLink(src); err != nil {
		return nil, err
	}

	// Upsert no-follow links without edges
	for _, dstlink := range payload.NoFollowLinks {
		dst := &graph.Link{URL: dstlink}
		if err := gu.g.UpsertLink(dst); err != nil {
			return nil, err
		}
	}

	// Upsert links and create edges. Keep track of time so we can drop stale edges
	// that have not been updated after this loop.
	removeEdgesOlderThan := time.Now()
	for _, dstlink := range payload.Links {
		dst := &graph.Link{URL: dstlink}

		if err := gu.g.UpsertLink(dst); err != nil {
			return nil, err
		}

		if err := gu.g.UpsertEdge(&graph.Edge{Src: src.ID, Dst: dst.ID}); err != nil {
			return nil, err
		}
	}

	// Drop stale edges that were not modified while upserting outgoing edges.
	if err := gu.g.RemoveStaleEdges(src.ID, removeEdgesOlderThan); err != nil {
		return nil, err
	}

	return p, nil
}
