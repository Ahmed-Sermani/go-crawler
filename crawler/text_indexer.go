package crawler

import (
	"context"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/Ahmed-Sermani/go-crawler/pipeline"
)

var _ pipeline.Processor = (*textIndexer)(nil)

type textIndexer struct {
	indexer Indexer
}

func newTextIndexer(indexer Indexer) *textIndexer {
	return &textIndexer{
		indexer: indexer,
	}
}

func (ti *textIndexer) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	doc := &indexer.Document{
		LinkID:    payload.LinkID,
		URL:       payload.URL,
		Title:     payload.Title,
		Content:   payload.TextContent,
		IndexedAt: time.Now(),
	}
	if err := ti.indexer.Index(doc); err != nil {
		return nil, err
	}
	return p, nil
}
