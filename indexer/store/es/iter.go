package es

import (
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/elastic/go-elasticsearch/v8"
)

type esIterator struct {
	es        *elasticsearch.Client
	searchReq *esQuery

	cumIdx uint64
	rsIdx  int
	rs     *esSearchRes

	latchedDoc *indexer.Document
	lastErr    error
}

func (it *esIterator) Close() error {
	it.es = nil
	it.searchReq = nil
	it.cumIdx = it.rs.Hits.Total.Count
	return nil
}

func (it *esIterator) Next() bool {
	if it.lastErr != nil || it.rs == nil || it.cumIdx >= it.rs.Hits.Total.Count {
		return false
	}

	// Do we need to fetch the next batch?
	if it.rsIdx >= len(it.rs.Hits.HitList) {
		it.searchReq.From = it.searchReq.From + batchSize
		res, err := doSearch(it.es, it.searchReq)
		if err != nil {
			it.lastErr = err
			return false
		}

		esRes := &esSearchRes{}
		if it.lastErr = unmarshalResponse(res, esRes); it.lastErr != nil {
			return false
		}
		it.rs = esRes

		it.rsIdx = 0
	}
	it.latchedDoc = mapESDoc(it.rs.Hits.HitList[it.rsIdx].DocSource)
	it.cumIdx++
	it.rsIdx++
	return true
}

func (it *esIterator) Error() error {
	return it.lastErr
}

func (it *esIterator) Document() *indexer.Document {
	return it.latchedDoc
}

func (it *esIterator) TotalCount() uint64 {
	return it.rs.Hits.Total.Count
}
