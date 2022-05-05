package cdb

import (
	"database/sql"

	"github.com/Ahmed-Sermani/search/graph"
	"golang.org/x/xerrors"
)

type linkIterator struct {
	rows        *sql.Rows
	lastErr     error
	latchedLink *graph.Link
}

func (i *linkIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	link := &graph.Link{}
	i.lastErr = i.rows.Scan(&link.ID, &link.URL, &link.RetreivedAt)
	if i.lastErr != nil {
		return false
	}
	link.RetreivedAt = link.RetreivedAt.UTC()
	i.latchedLink = link
	return true
}

func (i *linkIterator) Error() error {
	return i.lastErr
}

func (i *linkIterator) Close() error {
	if err := i.rows.Close(); err != nil {
		return xerrors.Errorf("link iter: %w", err)
	}
	return nil
}

func (i *linkIterator) Link() *graph.Link {
	return i.latchedLink
}

type edgeIterator struct {
	rows        *sql.Rows
	lastErr     error
	latchedEdge *graph.Edge
}

func (i *edgeIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	edge := &graph.Edge{}
	i.lastErr = i.rows.Scan(&edge.ID, &edge.Src, &edge.Dst, &edge.UpdatedAt)
	if i.lastErr != nil {
		return false
	}
	edge.UpdatedAt = edge.UpdatedAt.UTC()
	i.latchedEdge = edge
	return true
}

func (i *edgeIterator) Error() error {
	return i.lastErr
}

func (i *edgeIterator) Close() error {
	if err := i.rows.Close(); err != nil {
		return xerrors.Errorf("edge iter: %w", err)
	}
	return nil
}

func (i *edgeIterator) Edge() *graph.Edge {
	return i.latchedEdge
}
