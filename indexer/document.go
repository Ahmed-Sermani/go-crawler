package indexer

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	LinkID uuid.UUID
	URL    string
	// The Title and Content attributes correspond to the value of the <title> element
	// if the link points to an HTML page, whereas the Content attribute stores
	// the block of text that was extracted by the crawler when processing the link.
	Title   string
	Content string
	// indicates when a particular document was last indexed
	IndexedAt time.Time
	// track the score that will be assigned to each document by the PageRank calculator component
	PageRank float64
}
