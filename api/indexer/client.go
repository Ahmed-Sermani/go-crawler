package indexer

import (
	"context"
	"io"

	"github.com/Ahmed-Sermani/go-crawler/api/indexer/proto"
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

//go:generate mockgen -package mocks -destination mocks/mock.go github.com/Ahmed-Sermani/go-crawler/api/indexer/proto IndexerClient,Indexer_SearchClient

// IndexerClient provides an API compatible with the index.Indexer interface
// for accessing a text indexer instances exposed by a remote gRPC server.
type IndexerClient struct {
	ctx context.Context
	cli proto.IndexerClient
}

// NewIndexerClient returns a new client instance that implements a subset
// of the index.Indexer interface by delegating methods to an indexer instance
// exposed by a remote gRPC sever.
func NewIndexerClient(ctx context.Context, rpcClient proto.IndexerClient) *IndexerClient {
	return &IndexerClient{ctx: ctx, cli: rpcClient}
}

func (c *IndexerClient) Index(doc *indexer.Document) error {
	req := &proto.Document{
		LinkId:  doc.LinkID[:],
		Url:     doc.URL,
		Title:   doc.Title,
		Content: doc.Content,
	}
	res, err := c.cli.Index(c.ctx, req)
	if err != nil {
		return err
	}

	t, err := ptypes.Timestamp(res.IndexedAt)
	if err != nil {
		return xerrors.Errorf("unable to decode indexedAt attribute of document %q: %w", doc.LinkID, err)
	}

	doc.IndexedAt = t
	return nil
}

func (c *IndexerClient) UpdateScore(linkID uuid.UUID, score float64) error {
	req := &proto.UpdateScoreRequest{
		LinkId:        linkID[:],
		PageRankScore: score,
	}
	_, err := c.cli.UpdateScore(c.ctx, req)
	return err
}

func (c *IndexerClient) Search(query indexer.Query) (indexer.Iterator, error) {
	ctx, cancelFn := context.WithCancel(c.ctx)
	req := &proto.Query{
		Type:       proto.Query_Type(query.Type),
		Expression: query.Expr,
		Offset:     query.Offset,
	}
	stream, err := c.cli.Search(ctx, req)
	if err != nil {
		cancelFn()
		return nil, err
	}

	// Read result count
	res, err := stream.Recv()
	if err != nil {
		cancelFn()
		return nil, err
	} else if res.GetDoc() != nil {
		cancelFn()
		return nil, xerrors.Errorf("expected server to report the result count before sending any documents")
	}

	return &resultIterator{
		total:    res.GetDocCount(),
		stream:   stream,
		cancelFn: cancelFn,
	}, nil
}

type resultIterator struct {
	total   uint64
	stream  proto.Indexer_SearchClient
	next    *indexer.Document
	lastErr error

	cancelFn func()
}

func (it *resultIterator) Next() bool {
	res, err := it.stream.Recv()
	if err != nil {
		if err != io.EOF {
			it.lastErr = err
		}
		it.cancelFn()
		return false
	}

	resDoc := res.GetDoc()
	if resDoc == nil {
		it.cancelFn()
		it.lastErr = xerrors.Errorf("received nil document in search result list")
		return false
	}

	linkID := uuidFromBytes(resDoc.LinkId)

	t, err := ptypes.Timestamp(resDoc.IndexedAt)
	if err != nil {
		it.cancelFn()
		it.lastErr = xerrors.Errorf("unable to decode indexedAt attribute of document %q: %w", linkID, err)
		return false
	}

	it.next = &indexer.Document{
		LinkID:    linkID,
		URL:       resDoc.Url,
		Title:     resDoc.Title,
		Content:   resDoc.Content,
		IndexedAt: t,
	}
	return true
}

func (it *resultIterator) Error() error { return it.lastErr }

func (it *resultIterator) Document() *indexer.Document { return it.next }

func (it *resultIterator) TotalCount() uint64 { return it.total }

func (it *resultIterator) Close() error {
	it.cancelFn()
	return nil
}
