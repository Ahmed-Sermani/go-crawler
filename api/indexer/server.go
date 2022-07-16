package indexer

import (
	"context"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/api/indexer/proto"
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
)

var _ proto.IndexerServer = (*IndexerServer)(nil)

// IndexerServer provides a gRPC layer for indexing and querying documents.
type IndexerServer struct {
	proto.UnimplementedIndexerServer
	i indexer.Indexer
}

// NewIndexerServer creates a new server instance that uses the provided
// indexer as its backing store.
func NewIndexerServer(i indexer.Indexer) *IndexerServer {
	return &IndexerServer{i: i}
}

// Index inserts a new document to the index or updates the index entry for
// and existing document.
func (s *IndexerServer) Index(_ context.Context, req *proto.Document) (*proto.Document, error) {
	doc := &indexer.Document{
		LinkID:  uuidFromBytes(req.LinkId),
		URL:     req.Url,
		Title:   req.Title,
		Content: req.Content,
	}

	err := s.i.Index(doc)
	if err != nil {
		return nil, err
	}

	req.IndexedAt = timeToProto(doc.IndexedAt)
	return req, nil
}

func (s *IndexerServer) UpdateScore(_ context.Context, req *proto.UpdateScoreRequest) (*empty.Empty, error) {
	linkID := uuidFromBytes(req.LinkId)
	return new(empty.Empty), s.i.UpdateScore(linkID, req.PageRankScore)
}

// Search the index for a particular query and stream the results back to the
// client. The first response will include the total result count while all
// subsequent responses will include documents from the resultset.
func (s *IndexerServer) Search(req *proto.Query, w proto.Indexer_SearchServer) error {
	query := indexer.Query{
		Type:   indexer.QueryType(req.Type),
		Expr:   req.Expression,
		Offset: req.Offset,
	}

	it, err := s.i.Search(query)
	if err != nil {
		return err
	}

	// Send back the total document count
	countRes := &proto.QueryResult{
		Result: &proto.QueryResult_DocCount{DocCount: it.TotalCount()},
	}
	if err = w.Send(countRes); err != nil {
		_ = it.Close()
		return err
	}

	// Start streaming
	for it.Next() {
		doc := it.Document()
		res := proto.QueryResult{
			Result: &proto.QueryResult_Doc{
				Doc: &proto.Document{
					LinkId:    doc.LinkID[:],
					Url:       doc.URL,
					Title:     doc.Title,
					Content:   doc.Content,
					IndexedAt: timeToProto(doc.IndexedAt),
				},
			},
		}
		if err = w.SendMsg(&res); err != nil {
			_ = it.Close()
			return err
		}
	}

	if err = it.Error(); err != nil {
		_ = it.Close()
		return err
	}

	return it.Close()
}

func uuidFromBytes(b []byte) uuid.UUID {
	if len(b) != 16 {
		return uuid.Nil
	}

	var dst uuid.UUID
	copy(dst[:], b)
	return dst
}

func timeToProto(t time.Time) *timestamp.Timestamp {
	ts, _ := ptypes.TimestampProto(t)
	return ts
}
