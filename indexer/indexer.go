package indexer

import "github.com/google/uuid"

type Indexer interface {
	Index(doc *Document) error
	FindByID(linkID uuid.UUID) (*Document, error)
	Search(query Query) (Iterator, error)
	UpdateScore(linkID uuid.UUID, score float64) error
}

type Iterator interface {
	Close() error
	Next() bool
	Error() error
	Document() *Document
	TotalCount() uint64
}

type Query struct {
	Type   QueryType
	Expr   string // stores the search query that's entered by the end user
	Offset uint64
}

type QueryType uint8

const (
	Match QueryType = iota
	Phrase
)
