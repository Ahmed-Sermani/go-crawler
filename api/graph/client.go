package graph

import (
	"context"
	"io"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/api/graph/proto"
	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
)

//go:generate mockgen -package mocks -destination mocks/mock.go github.com/Ahmed-Sermani/go-crawler/api/graph/proto GraphClient,Graph_LinksClient,Graph_EdgesClient

// GraphClient provides an API compatible with the graph.Graph interface
// for accessing graph instances exposed by a remote gRPC server.
type GraphClient struct {
	ctx context.Context
	cli proto.GraphClient
}

// NewGraphClient returns a new client instance that implements a subset
// of the graph.Graph interface by delegating methods to a graph instance
// exposed by a remote gRPC server.
func NewGraphClient(ctx context.Context, rpcClient proto.GraphClient) *GraphClient {
	return &GraphClient{ctx: ctx, cli: rpcClient}
}

func (c *GraphClient) UpsertLink(link *graph.Link) error {
	req := &proto.Link{
		Uuid:        link.ID[:],
		Url:         link.URL,
		RetrievedAt: timeToProto(link.RetreivedAt),
	}
	res, err := c.cli.UpsertLink(c.ctx, req)
	if err != nil {
		return err
	}

	link.ID = uuidFromBytes(res.Uuid)
	link.URL = res.Url
	if link.RetreivedAt, err = ptypes.Timestamp(res.RetrievedAt); err != nil {
		return err
	}

	return nil
}

func (c *GraphClient) UpsertEdge(edge *graph.Edge) error {
	req := &proto.Edge{
		Uuid:    edge.ID[:],
		SrcUuid: edge.Src[:],
		DstUuid: edge.Dst[:],
	}
	res, err := c.cli.UpsertEdge(c.ctx, req)
	if err != nil {
		return err
	}

	edge.ID = uuidFromBytes(res.Uuid)
	if edge.UpdatedAt, err = ptypes.Timestamp(res.UpdatedAt); err != nil {
		return err
	}

	return nil
}

func (c *GraphClient) Links(fromID, toID uuid.UUID, accessedBefore time.Time) (graph.LinkIterator, error) {
	filter, err := ptypes.TimestampProto(accessedBefore)
	if err != nil {
		return nil, err
	}

	req := &proto.Range{
		FromUuid: fromID[:],
		ToUuid:   toID[:],
		Filter:   filter,
	}

	ctx, cancelFn := context.WithCancel(c.ctx)
	stream, err := c.cli.Links(ctx, req)
	if err != nil {
		cancelFn()
		return nil, err
	}

	return &linkIterator{stream: stream, cancelFn: cancelFn}, nil
}

func (c *GraphClient) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (graph.EdgeIterator, error) {
	filter, err := ptypes.TimestampProto(updatedBefore)
	if err != nil {
		return nil, err
	}

	req := &proto.Range{
		FromUuid: fromID[:],
		ToUuid:   toID[:],
		Filter:   filter,
	}

	ctx, cancelFn := context.WithCancel(c.ctx)
	stream, err := c.cli.Edges(ctx, req)
	if err != nil {
		cancelFn()
		return nil, err
	}

	return &edgeIterator{stream: stream, cancelFn: cancelFn}, nil
}

func (c *GraphClient) RemoveStaleEdges(from uuid.UUID, updatedBefore time.Time) error {
	req := &proto.RemoveStaleEdgesQuery{
		FromUuid:      from[:],
		UpdatedBefore: timeToProto(updatedBefore),
	}

	_, err := c.cli.RemoveStaleEdges(c.ctx, req)
	return err
}

type linkIterator struct {
	stream  proto.Graph_LinksClient
	next    *graph.Link
	lastErr error

	cancelFn func()
}

func (it *linkIterator) Next() bool {
	res, err := it.stream.Recv()
	if err != nil {
		if err != io.EOF {
			it.lastErr = err
		}
		it.cancelFn()
		return false
	}

	lastAccessed, err := ptypes.Timestamp(res.RetrievedAt)
	if err != nil {
		it.lastErr = err
		it.cancelFn()
		return false
	}

	it.next = &graph.Link{
		ID:          uuidFromBytes(res.Uuid),
		URL:         res.Url,
		RetreivedAt: lastAccessed,
	}
	return true
}

func (it *linkIterator) Error() error { return it.lastErr }

func (it *linkIterator) Link() *graph.Link { return it.next }

func (it *linkIterator) Close() error {
	it.cancelFn()
	return nil
}

type edgeIterator struct {
	stream  proto.Graph_EdgesClient
	next    *graph.Edge
	lastErr error

	cancelFn func()
}

func (it *edgeIterator) Next() bool {
	res, err := it.stream.Recv()
	if err != nil {
		if err != io.EOF {
			it.lastErr = err
		}
		it.cancelFn()
		return false
	}

	updatedAt, err := ptypes.Timestamp(res.UpdatedAt)
	if err != nil {
		it.lastErr = err
		it.cancelFn()
		return false
	}

	it.next = &graph.Edge{
		ID:        uuidFromBytes(res.Uuid),
		Src:       uuidFromBytes(res.SrcUuid),
		Dst:       uuidFromBytes(res.DstUuid),
		UpdatedAt: updatedAt,
	}
	return true
}

func (it *edgeIterator) Error() error { return it.lastErr }

func (it *edgeIterator) Edge() *graph.Edge { return it.next }

func (it *edgeIterator) Close() error {
	it.cancelFn()
	return nil
}
