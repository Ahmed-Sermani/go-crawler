package cdb

import (
	"database/sql"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/xerrors"
)

const (
	upsertLinkQuery = `
  INSERT INTO links (url, retrieved_at) VALUES ($1, $2)
  ON CONFLICT (url) DO UPDATE SET retrieved_at=GREATEST(links.retrieved_at, $2)
  RETURNING id, retrieved_at
  `
	upsertEdgeQuery = `
  INSERT INTO edges (src, dst, updated_at) VALUES ($1, $2, NOW())
  ON CONFLICT (src, dst) DO UPDATE SET updated_at=NOW()
  RETURNING id, updated_at
  `
	getLinkQuery = `
  SELECT url, retrieved_at FROM links WHERE id=$1
  `
	iterLinkQuery = `
  SELECT id, url, retrieved_at FROM links WHERE id >= $1 AND id < $2 AND retrieved_at < $3
  `
	iterEdgesQuery = `
  SELECT id, src, dst, updated_at FROM edges WHERE src >= $1 AND src < $2 AND updated_at < $3
  `
	rmStaleEdgesQuery = `
  DELETE FROM edges WHERE src=$1 AND updated_at < $2
  `
)

type CockroachDBGraph struct {
	db *sql.DB
}

func NewCockroachDBGraph(dsn string) (*CockroachDBGraph, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return &CockroachDBGraph{db}, nil
}

func (c *CockroachDBGraph) Close() error {
	return c.db.Close()
}

func (c *CockroachDBGraph) UpsertLink(link *graph.Link) error {
	row := c.db.QueryRow(upsertLinkQuery, link.URL, link.RetreivedAt.UTC())
	if err := row.Scan(&link.ID, &link.RetreivedAt); err != nil {
		return xerrors.Errorf("upsert link: %w", err)
	}
	link.RetreivedAt = link.RetreivedAt.UTC()
	return nil
}

func (c *CockroachDBGraph) UpsertEdge(edge *graph.Edge) error {
	row := c.db.QueryRow(upsertEdgeQuery, edge.Src, edge.Dst)
	if err := row.Scan(&edge.ID, &edge.UpdatedAt); err != nil {
		if isForeginKeyError(err) {
			err = graph.ErrUnknownEdgeLinks
		}
		return xerrors.Errorf("upsert edge: %w", err)
	}
	edge.UpdatedAt = edge.UpdatedAt.UTC()
	return nil
}

func isForeginKeyError(err error) bool {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return false
	}

	return pqErr.Code.Name() == "foreign_key_violation"
}

func (c *CockroachDBGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	row := c.db.QueryRow(getLinkQuery, id)
	link := &graph.Link{ID: id}
	if err := row.Scan(&link.URL, &link.RetreivedAt); err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			return nil, xerrors.Errorf("find link: %w", graph.ErrNotFound)
		}
		return nil, xerrors.Errorf("find link: %w", err)
	}
	link.RetreivedAt = link.RetreivedAt.UTC()
	return link, nil
}

func (c *CockroachDBGraph) Links(fromID, toID uuid.UUID, accessedBefore time.Time) (graph.LinkIterator, error) {
	rows, err := c.db.Query(iterLinkQuery, fromID, toID, accessedBefore.UTC())
	if err != nil {
		return nil, xerrors.Errorf("links: %w", err)
	}
	return &linkIterator{rows: rows}, nil
}

func (c *CockroachDBGraph) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (graph.EdgeIterator, error) {
	rows, err := c.db.Query(iterEdgesQuery, fromID, toID, updatedBefore.UTC())
	if err != nil {
		return nil, xerrors.Errorf("edges: %w", err)
	}

	return &edgeIterator{rows: rows}, nil
}

func (c *CockroachDBGraph) RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error {
	_, err := c.db.Exec(rmStaleEdgesQuery, fromID, updatedBefore.UTC())
	if err != nil {
		return xerrors.Errorf("remove stale edges: %w", err)
	}
	return nil
}
