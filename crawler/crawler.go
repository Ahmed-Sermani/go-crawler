/*
   Manages the crawling of links
*/

package crawler

import (
	"net/http"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/google/uuid"
)

// URLGetter is implemented by objects that can perform HTTP GET requests.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector is implemented by objects that can detect whether a
// host resolves to a private network address.
// used as a secuity machnism to prevent exposing internal services to the crawler
type PrivateNetworkDetector interface {
	IsPrivate(host string) (bool, error)
}

// Graph is implemented by objects that can upsert links and edges into a link
// graph instance.
type Graph interface {
	UpsertLink(link *graph.Link) error
	UpsertEdge(edge *graph.Edge) error

	// RemoveStaleEdges removes any edge that originates from the specified
	// link ID and was updated before the specified timestamp.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}

// Indexer is implemented by objects that can index the contents of web-pages
// retrieved by the crawler pipeline.
type Indexer interface {
	Index(doc *indexer.Document) error
}
