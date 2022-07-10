package memory

import (
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/blevesearch/bleve"
)

type memIterator struct {
	idx       *InMemoryIndexer
	searchReq *bleve.SearchRequest

	cumIdx uint64
	resIdx int
	res    *bleve.SearchResult

	latchedDoc *indexer.Document
	lastErr    error
}

func (iter *memIterator) Next() bool {
	if iter.lastErr != nil || iter.res == nil || iter.cumIdx >= iter.res.Total {
		return false
	}

	// Do we need to fetch the next batch?
	if iter.resIdx >= iter.res.Hits.Len() {
		iter.searchReq.From += iter.searchReq.Size
		if iter.res, iter.lastErr = iter.idx.idx.Search(iter.searchReq); iter.lastErr != nil {
			return false
		}
		iter.resIdx = 0
	}
	nextID := iter.res.Hits[iter.resIdx].ID
	if iter.latchedDoc, iter.lastErr = iter.idx.findByID(nextID); iter.lastErr != nil {
		return false
	}

	iter.cumIdx++
	iter.resIdx++
	return true
}

func (iter *memIterator) Close() error {
	iter.idx = nil
	iter.searchReq = nil
	if iter.res != nil {
		iter.cumIdx = iter.res.Total
	}
	return nil
}

func (iter *memIterator) Document() *indexer.Document {
	return iter.latchedDoc
}

func (iter *memIterator) Error() error {
	return iter.lastErr
}

func (iter *memIterator) TotalCount() uint64 {
	if iter.res == nil {
		return 0
	}
	return iter.res.Total
}
