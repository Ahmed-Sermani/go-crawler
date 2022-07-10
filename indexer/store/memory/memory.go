package memory

import (
	"sync"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

const defaultBatchSize = 10

type InMemoryIndexer struct {
	mu        sync.RWMutex
	documents map[string]*indexer.Document

	idx bleve.Index
}

type memDoc struct {
	Title    string
	Content  string
	PageRank float64
}

func NewInMemoryBleveIndexer() (*InMemoryIndexer, error) {
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}
	return &InMemoryIndexer{
		idx:       idx,
		documents: make(map[string]*indexer.Document),
	}, nil
}

func (i *InMemoryIndexer) Index(doc *indexer.Document) error {
	if doc.LinkID == uuid.Nil {
		return xerrors.Errorf("index: %w", indexer.ErrMissingLinkID)
	}
	doc.IndexedAt = time.Now()
	dcopy := cpDoc(doc)
	k := dcopy.LinkID.String()
	i.mu.Lock()
	defer i.mu.Unlock()

	if orig, ok := i.documents[k]; ok {
		dcopy.PageRank = orig.PageRank
	}

	if err := i.idx.Index(k, makeMemDoc(dcopy)); err != nil {
		return xerrors.Errorf("index: %w", err)
	}
	i.documents[k] = dcopy
	return nil
}

func (i *InMemoryIndexer) FindByID(linkID uuid.UUID) (*indexer.Document, error) {
	return i.findByID(linkID.String())
}

func (i *InMemoryIndexer) UpdateScore(linkID uuid.UUID, score float64) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	k := linkID.String()
	doc, found := i.documents[k]
	if !found {
		doc = &indexer.Document{LinkID: linkID}
		i.documents[k] = doc
	}
	doc.PageRank = score
	if err := i.idx.Index(k, makeMemDoc(doc)); err != nil {
		return xerrors.Errorf("update score: %w", err)
	}
	return nil
}

func (i *InMemoryIndexer) Search(sq indexer.Query) (indexer.Iterator, error) {
	var query query.Query
	switch sq.Type {
	case indexer.Phrase:
		query = bleve.NewMatchPhraseQuery(sq.Expr)
	default:
		query = bleve.NewMatchQuery(sq.Expr)
	}
	searchReq := bleve.NewSearchRequest(query)
	searchReq.SortBy([]string{"-PageRank", "-_score"})
	searchReq.Size = defaultBatchSize
	searchReq.From = int(sq.Offset)
	res, err := i.idx.Search(searchReq)
	if err != nil {
		return nil, xerrors.Errorf("search: %w", err)
	}
	return &memIterator{idx: i, searchReq: searchReq, res: res, cumIdx: sq.Offset}, nil
}

func (i *InMemoryIndexer) Close() error {
	return i.idx.Close()
}

func (i *InMemoryIndexer) findByID(id string) (*indexer.Document, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if d, found := i.documents[id]; found {
		return cpDoc(d), nil
	}
	return nil, xerrors.Errorf("find by id: %w", indexer.ErrNotFound)
}

func cpDoc(d *indexer.Document) *indexer.Document {
	dcopy := new(indexer.Document)
	*dcopy = *d
	return dcopy
}

func makeMemDoc(d *indexer.Document) memDoc {
	return memDoc{
		Title:    d.Title,
		Content:  d.Content,
		PageRank: d.PageRank,
	}
}
