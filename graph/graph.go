package graph

import (
	"time"

	"github.com/google/uuid"
)

type Iterator interface {
	Next() bool
	Error() error
	Close() error
}

type Link struct {
	ID          uuid.UUID
	URL         string
	RetreivedAt time.Time
}

type LinkIterator interface {
	Iterator
	Link() *Link
}

type Edge struct {
	ID        uuid.UUID
	Src       uuid.UUID
	Dst       uuid.UUID
	UpdatedAt time.Time
}

type EdgeIterator interface {
	Iterator
	Edge() *Edge
}

type Graph interface {
	UpsertLink(*Link) error
	FindLink(uuid.UUID) (*Link, error)

	UpsertEdge(*Edge) error
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error

	Links(fromID, toID uuid.UUID, retreviedBefore time.Time) (LinkIterator, error)
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (EdgeIterator, error)
}
