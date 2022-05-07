package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Ahmed-Sermani/search/indexer"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

const indexName = "txtindexer"

const batchSize = 10

type esErrorRes struct {
	Err struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
}

func (e esErrorRes) Error() string {
	return fmt.Sprintf("%s: %s", e.Err.Type, e.Err.Reason)
}

type esDoc struct {
	LinkID    string    `json:"LinkID"`
	URL       string    `json:"URL"`
	Title     string    `json:"Title"`
	Content   string    `json:"Content"`
	IndexedAt time.Time `json:"IndexedAt"`
	PageRank  float64   `json:"PageRank,omitempty"`
}

type esQuery struct {
	Query struct {
		ScoreFunc struct {
			Query struct {
				MultiMatch struct {
					Type   string   `json:"type"`
					Query  string   `json:"query"`
					Fields []string `json:"fields"`
				} `json:"multi_match"`
			} `json:"query"`
			ScoreScript struct {
				Script struct {
					Source string `json:"source"`
				} `json:"script"`
			} `json:"script_score"`
		} `json:"function_score"`
	} `json:"query"`
	From int `json:"from"`
	To   int `json:"to"`
}

type esSearchRes struct {
	Hits struct {
		Total struct {
			Count uint64 `json:"value"`
		} `json:"total"`
		HitList []struct {
			DocSource esDoc `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type ESIndexer struct {
	es *elasticsearch.Client
}

func NewESIndexer(nodes []string, syncUpdates bool) (*ESIndexer, error) {
	cfg := elasticsearch.Config{
		Addresses: nodes,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if err = ensureIndex(es); err != nil {
		return nil, err
	}
	return &ESIndexer{es: es}, nil

}

func ensureIndex(es *elasticsearch.Client) error {
	const mapping = `
	{
		"mappings" : {
		  "properties": {
			"LinkID": {"type": "keyword"},
			"URL": {"type": "keyword"},
			"Content": {"type": "text"},
			"Title": {"type": "text"},
			"IndexedAt": {"type": "date"},
			"PageRank": {"type": "double"}
		  }
		}
	}`

	mappingReader := strings.NewReader(mapping)
	res, err := es.Indices.Create(indexName, es.Indices.Create.WithBody(mappingReader))
	if err != nil {
		return xerrors.Errorf("create index error: %w", err)
	} else if res.IsError() {
		defer res.Body.Close()
		var esErr esErrorRes
		if err := json.NewDecoder(res.Body).Decode(&esErr); err != nil {
			return err
		}
		if esErr.Err.Type == "resource_already_exists_exception" {
			return nil
		}
		return xerrors.Errorf("create index: %w", esErr)
	}
	return nil
}

func (i *ESIndexer) Index(doc *indexer.Document) error {
	if doc.LinkID == uuid.Nil {
		return xerrors.Errorf("index: %w", indexer.ErrMissingLinkID)
	}
	var buf bytes.Buffer
	esDoc := makeESDoc(doc)
	update := map[string]interface{}{
		"doc":           esDoc,
		"doc_as_upsert": true,
	}
	if err := json.NewEncoder(&buf).Encode(update); err != nil {
		return xerrors.Errorf("index: %w", err)
	}
	res, err := i.es.Update(indexName, esDoc.LinkID, &buf)
	if err != nil {
		return xerrors.Errorf("index: %w", err)
	}

	esUpdateResponse := struct {
		Result string `json:"result"`
	}{}

	if err := unmarshalResponse(res, &esUpdateResponse); err != nil {
		return xerrors.Errorf("index: %w", err)
	}
	return nil
}

func (i *ESIndexer) FindByID(linkID uuid.UUID) (*indexer.Document, error) {
	res, err := i.es.GetSource(indexName, linkID.String())
	if err != nil {
		return nil, xerrors.Errorf("find by ID: %w", err)
	}
	var esRes struct {
		DocSource esDoc `json:"_source"`
	}

	if err := unmarshalResponse(res, &esRes); err != nil {
		return nil, xerrors.Errorf("find by ID: %w", err)
	}

	return mapESDoc(esRes.DocSource), nil
}

func (i *ESIndexer) Search(q indexer.Query) (indexer.Iterator, error) {
	var qtype string
	switch q.Type {
	case indexer.Phrase:
		qtype = "phrase"
	default:
		qtype = "best_fields"
	}

	var esQuery esQuery
	esQuery.From = int(q.Offset)
	esQuery.To = batchSize
	esQuery.Query.ScoreFunc.Query.MultiMatch.Fields = []string{"Title", "Content"}
	esQuery.Query.ScoreFunc.Query.MultiMatch.Query = q.Expr
	esQuery.Query.ScoreFunc.ScoreScript.Script.Source = "_score + doc['PageRank'].value"
	esQuery.Query.ScoreFunc.Query.MultiMatch.Type = qtype

	res, err := doSearch(i.es, &esQuery)
	if err != nil {
		return nil, xerrors.Errorf("search: %w", err)
	}

	var esRes esSearchRes
	if err = unmarshalResponse(res, &esRes); err != nil {
		return nil, xerrors.Errorf("search: %w", err)
	}

	return &esIterator{es: i.es, searchReq: &esQuery, rs: &esRes, cumIdx: q.Offset}, nil
}

func doSearch(es *elasticsearch.Client, query *esQuery) (*esapi.Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, xerrors.Errorf("search: %w", err)
	}

	return es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(indexName),
		es.Search.WithBody(&buf),
	)
}
func makeESDoc(d *indexer.Document) esDoc {
	return esDoc{
		LinkID:    d.LinkID.String(),
		URL:       d.URL,
		Title:     d.Title,
		Content:   d.Content,
		IndexedAt: d.IndexedAt.UTC(),
	}
}

func unmarshalResponse(res *esapi.Response, to interface{}) error {
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		var esErr esErrorRes
		if err := json.NewDecoder(res.Body).Decode(&esErr); err != nil {
			return err
		}
		return xerrors.Errorf("create index: %w", esErr)
	}
	return json.NewDecoder(res.Body).Decode(to)
}

func mapESDoc(doc esDoc) *indexer.Document {
	return &indexer.Document{
		LinkID:    uuid.MustParse(doc.LinkID),
		URL:       doc.URL,
		Title:     doc.Title,
		Content:   doc.Content,
		IndexedAt: doc.IndexedAt.UTC(),
		PageRank:  doc.PageRank,
	}
}
