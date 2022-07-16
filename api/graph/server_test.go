package graph

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/api/graph/proto"
	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/Ahmed-Sermani/go-crawler/graph/store/memory"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(ServerTestSuite))
var minUUID = uuid.Nil
var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

type ServerTestSuite struct {
	g graph.Graph

	netListener *bufconn.Listener
	grpcSrv     *grpc.Server

	cliConn *grpc.ClientConn
	cli     proto.GraphClient
}

func (s *ServerTestSuite) SetUpTest(c *gc.C) {
	s.g = memory.NewInMemoryGraph()

	s.netListener = bufconn.Listen(1024)
	s.grpcSrv = grpc.NewServer()
	proto.RegisterGraphServer(s.grpcSrv, NewGraphServer(s.g))
	go func() {
		err := s.grpcSrv.Serve(s.netListener)
		c.Assert(err, gc.IsNil)
	}()

	var err error
	s.cliConn, err = grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return s.netListener.Dial() }),
		grpc.WithInsecure(),
	)
	c.Assert(err, gc.IsNil)
	s.cli = proto.NewGraphClient(s.cliConn)
}

func (s *ServerTestSuite) TearDownTest(c *gc.C) {
	_ = s.cliConn.Close()
	s.grpcSrv.Stop()
	_ = s.netListener.Close()
}

func (s *ServerTestSuite) TestInsertLink(c *gc.C) {
	now := time.Now().Truncate(time.Second).UTC()
	req := &proto.Link{
		Url:         "http://example.com",
		RetrievedAt: mustEncodeTimestamp(c, now),
	}

	res, err := s.cli.UpsertLink(context.TODO(), req)
	c.Assert(err, gc.IsNil)
	c.Assert(res.Uuid, gc.Not(gc.DeepEquals), req.Uuid, gc.Commentf("UUID not assigned to new link"))
	c.Assert(mustDecodeTimestamp(c, res.RetrievedAt), gc.Equals, now)
}

func (s *ServerTestSuite) TestUpdateLink(c *gc.C) {
	link := &graph.Link{URL: "http://example.com"}
	c.Assert(s.g.UpsertLink(link), gc.IsNil)

	now := time.Now().Truncate(time.Second).UTC()
	req := &proto.Link{
		Uuid:        link.ID[:],
		Url:         "http://example.com",
		RetrievedAt: mustEncodeTimestamp(c, now),
	}

	res, err := s.cli.UpsertLink(context.TODO(), req)
	c.Assert(err, gc.IsNil)
	c.Assert(res.Uuid, gc.DeepEquals, link.ID[:], gc.Commentf("UUID for existing link modified"))
	c.Assert(res.Url, gc.Equals, req.Url, gc.Commentf("URL not updated"))
	c.Assert(mustDecodeTimestamp(c, res.RetrievedAt), gc.Equals, now)
}

func (s *ServerTestSuite) TestInsertEdge(c *gc.C) {
	src := &graph.Link{URL: "http://example.com"}
	dst := &graph.Link{URL: "http://foo.com"}
	c.Assert(s.g.UpsertLink(src), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst), gc.IsNil)

	req := &proto.Edge{
		SrcUuid: src.ID[:],
		DstUuid: dst.ID[:],
	}
	res, err := s.cli.UpsertEdge(context.TODO(), req)
	c.Assert(err, gc.IsNil)
	c.Assert(res.Uuid, gc.Not(gc.DeepEquals), req.Uuid, gc.Commentf("UUID not assigned to new edge"))
	c.Assert(res.SrcUuid[:], gc.DeepEquals, req.SrcUuid[:])
	c.Assert(res.DstUuid[:], gc.DeepEquals, req.DstUuid[:])
	c.Assert(res.UpdatedAt.Seconds, gc.Not(gc.Equals), 0)
}

func (s *ServerTestSuite) TestUpdateEdge(c *gc.C) {
	src := &graph.Link{URL: "http://example.com"}
	dst := &graph.Link{URL: "http://foo.com"}
	c.Assert(s.g.UpsertLink(src), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst), gc.IsNil)
	edge := &graph.Edge{Src: src.ID, Dst: dst.ID}
	c.Assert(s.g.UpsertEdge(edge), gc.IsNil)

	req := &proto.Edge{
		Uuid:    edge.ID[:],
		SrcUuid: src.ID[:],
		DstUuid: dst.ID[:],
	}

	res, err := s.cli.UpsertEdge(context.TODO(), req)
	c.Assert(err, gc.IsNil)
	c.Assert(res.Uuid, gc.DeepEquals, req.Uuid, gc.Commentf("UUID for existing edge modified"))
	c.Assert(res.SrcUuid[:], gc.DeepEquals, req.SrcUuid[:])
	c.Assert(res.DstUuid[:], gc.DeepEquals, req.DstUuid[:])
	c.Assert(mustDecodeTimestamp(c, res.UpdatedAt).After(edge.UpdatedAt), gc.Equals, true, gc.Commentf("expected UpdatedAt field to be set to a newer timestamp after updating edge"))
}

func (s *ServerTestSuite) TestLinks(c *gc.C) {
	sawLinks := make(map[uuid.UUID]bool)
	for i := 0; i < 100; i++ {
		link := &graph.Link{
			URL: fmt.Sprintf("http://example.com/%d", i),
		}
		c.Assert(s.g.UpsertLink(link), gc.IsNil)
		sawLinks[link.ID] = false
	}

	filter := mustEncodeTimestamp(c, time.Now().Add(time.Hour))
	stream, err := s.cli.Links(context.TODO(), &proto.Range{FromUuid: minUUID[:], ToUuid: maxUUID[:], Filter: filter})
	c.Assert(err, gc.IsNil)
	for {
		next, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Fatal(err)
		}

		linkID, err := uuid.FromBytes(next.Uuid)
		c.Assert(err, gc.IsNil)

		alreadySeen, exists := sawLinks[linkID]
		if !exists {
			c.Fatalf("saw unexpected link with ID %q", linkID)
		} else if alreadySeen {
			c.Fatalf("saw duplicate link with ID %q", linkID)
		}
		sawLinks[linkID] = true
	}

	for linkID, seen := range sawLinks {
		if !seen {
			c.Fatalf("expected to see link with ID %q", linkID)
		}
	}
}

func (s *ServerTestSuite) TestEdges(c *gc.C) {
	links := make([]uuid.UUID, 100)
	for i := 0; i < len(links); i++ {
		link := &graph.Link{
			URL: fmt.Sprintf("http://example.com/%d", i),
		}
		c.Assert(s.g.UpsertLink(link), gc.IsNil)
		links[i] = link.ID
	}
	sawEdges := make(map[uuid.UUID]bool)
	for i := 0; i < len(links); i++ {
		edge := &graph.Edge{
			Src: links[0],
			Dst: links[i],
		}
		c.Assert(s.g.UpsertEdge(edge), gc.IsNil)
		sawEdges[edge.ID] = false
	}

	filter := mustEncodeTimestamp(c, time.Now().Add(time.Hour))
	stream, err := s.cli.Edges(context.TODO(), &proto.Range{FromUuid: minUUID[:], ToUuid: maxUUID[:], Filter: filter})
	c.Assert(err, gc.IsNil)
	for {
		next, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Fatal(err)
		}

		edgeID, err := uuid.FromBytes(next.Uuid)
		c.Assert(err, gc.IsNil)

		alreadySeen, exists := sawEdges[edgeID]
		if !exists {
			c.Fatalf("saw unexpected edges with ID %q", edgeID)
		} else if alreadySeen {
			c.Fatalf("saw duplicate edges with ID %q", edgeID)
		}
		sawEdges[edgeID] = true
	}

	for edgeID, seen := range sawEdges {
		if !seen {
			c.Fatalf("expected to see edge with ID %q", edgeID)
		}
	}
}

func (s *ServerTestSuite) TestRetainVersionedEdges(c *gc.C) {
	src := &graph.Link{URL: "http://example.com"}
	dst1 := &graph.Link{URL: "http://foo.com"}
	dst2 := &graph.Link{URL: "http://bar.com"}
	c.Assert(s.g.UpsertLink(src), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst1), gc.IsNil)
	c.Assert(s.g.UpsertLink(dst2), gc.IsNil)

	edge1 := &graph.Edge{Src: src.ID, Dst: dst1.ID}
	c.Assert(s.g.UpsertEdge(edge1), gc.IsNil)

	time.Sleep(100 * time.Millisecond)
	t1 := time.Now()
	edge2 := &graph.Edge{Src: src.ID, Dst: dst2.ID}
	c.Assert(s.g.UpsertEdge(edge2), gc.IsNil)

	req := &proto.RemoveStaleEdgesQuery{
		FromUuid:      src.ID[:],
		UpdatedBefore: mustEncodeTimestamp(c, t1),
	}
	_, err := s.cli.RemoveStaleEdges(context.TODO(), req)
	c.Assert(err, gc.IsNil)

	it, err := s.g.Edges(minUUID, maxUUID, time.Now())
	c.Assert(err, gc.IsNil)

	var edgeCount int
	for it.Next() {
		edgeCount++
		edge := it.Edge()
		c.Assert(edge.ID, gc.Not(gc.DeepEquals), edge1.ID, gc.Commentf("expected edge1 to be dropper"))
	}
	c.Assert(it.Error(), gc.IsNil)
	c.Assert(edgeCount, gc.Equals, 1)
}

func Test(t *testing.T) {
	gc.TestingT(t)
}

func mustEncodeTimestamp(c *gc.C, t time.Time) *timestamp.Timestamp {
	ts, err := ptypes.TimestampProto(t)
	c.Assert(err, gc.IsNil)
	return ts
}

func mustDecodeTimestamp(c *gc.C, ts *timestamp.Timestamp) time.Time {
	t, err := ptypes.Timestamp(ts)
	c.Assert(err, gc.IsNil)
	return t
}
