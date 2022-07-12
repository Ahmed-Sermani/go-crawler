/*
   Manages the crawling of links
*/

package crawler

import (
	"context"
	"net/http"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/Ahmed-Sermani/go-crawler/pipeline"
	"github.com/Ahmed-Sermani/go-crawler/pipeline/runners"
	"github.com/google/uuid"
)

//go:generate mockgen -package mocks -destination mocks/mocks.go github.com/Ahmed-Sermani/go-crawler/crawler URLGetter,PrivateNetworkDetector,Graph,Indexer

// Crawler implements a web-page crawling pipeline consisting of the following
// stages:
//
// - Given a URL, retrieve the web-page contents from the remote server.
// - Extract and resolve absolute and relative links from the retrieved page.
// - Extract page title and text content from the retrieved page.
// - Update the link graph: add new links and create edges between the crawled
//   page and the links within it.
// - Index crawled page title and text content.
type Crawler struct {
	pipeline *pipeline.Pipeline
}

func NewCrawler(cfg Config) *Crawler {
	return &Crawler{
		pipeline: assembleCrawlerPipeline(cfg),
	}
}

// Crawl iterates linkIter and sends each link through the crawler pipeline
// returning the total count of links that went through the pipeline. Calls to
// Crawl block until the link iterator is exhausted, an error occurs or the
// context is cancelled.
// It's safe to be called by concurrent gorutines.
func (c *Crawler) Crawl(ctx context.Context, linkIter graph.LinkIterator) (int, error) {
	sink := &countingSink{}
	source := &linkSource{linkIter: linkIter}
	err := c.pipeline.Process(ctx, source, sink)
	return sink.getCount(), err
}

func assembleCrawlerPipeline(cfg Config) *pipeline.Pipeline {
	return pipeline.New(
		runners.FixedWorkerPool(
			newLinkFetcher(cfg.URLGetter, cfg.PrivateNetworkDetector),
			cfg.FetchWorkers,
		),
		runners.FIFO(newLinkExtractor(cfg.PrivateNetworkDetector)),
		runners.FIFO(newTextExtractor()),
		runners.Broadcast(
			newGraphUpdater(cfg.Graph),
			newTextIndexer(cfg.Indexer),
		),
	)
}

type Config struct {
	PrivateNetworkDetector PrivateNetworkDetector
	URLGetter              URLGetter
	Graph                  Graph
	Indexer                Indexer
	FetchWorkers           int
}

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
