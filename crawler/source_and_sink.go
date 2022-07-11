package crawler

import (
	"context"

	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/Ahmed-Sermani/go-crawler/pipeline"
)

type linkSource struct {
	linkIter graph.LinkIterator
}

func (ls *linkSource) Error() error                  { return ls.linkIter.Error() }
func (ls *linkSource) Next(ctx context.Context) bool { return ls.linkIter.Next() }

func (ls *linkSource) Payload() pipeline.Payload {
	link := ls.linkIter.Link()
	payload := payloadPool.Get().(*crawlerPayload)
	payload.LinkID = link.ID
	payload.URL = link.URL
	payload.RetrievedAt = link.RetreivedAt
	return payload
}

type nopSink struct{}

func (nopSink) Consume(context.Context, pipeline.Payload) error {
	return nil
}

type countingSink struct {
	count int
}

func (s *countingSink) Consume(_ context.Context, p pipeline.Payload) error {
	s.count++
	return nil
}

func (s *countingSink) getCount() int {
	return s.count
}
