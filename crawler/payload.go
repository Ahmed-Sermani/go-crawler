package crawler

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
	"github.com/google/uuid"
)

var (
	_ pipeline.Payload = (*crawlerPayload)(nil)

	// since the crawler works with large number of payloads concurrently,
	// this may result in more pressuce on the GC and hence more GC pauses.
	// to overcome this issue this maintain a pool of never garbage collected objects
	// that are reused by the crawler. Each request to the pool may return an existing object or creates a new one.
	payloadPool = sync.Pool{
		New: func() any { return new(crawlerPayload) },
	}
)

type crawlerPayload struct {
	LinkID      uuid.UUID
	URL         string
	RetrievedAt time.Time

	RawContent bytes.Buffer

	NoFollowLinks []string

	Links       []string
	Title       string
	TextContent string
}

func (p *crawlerPayload) Clone() pipeline.Payload {
	newp := payloadPool.Get().(*crawlerPayload)
	newp.LinkID = p.LinkID
	newp.URL = p.URL
	newp.RetrievedAt = p.RetrievedAt
	newp.NoFollowLinks = append([]string(nil), p.NoFollowLinks...)
	newp.Links = append([]string(nil), p.Links...)
	newp.Title = p.Title
	newp.TextContent = p.TextContent

	_, err := io.Copy(&newp.RawContent, &p.RawContent)
	if err != nil {
		panic(fmt.Sprintf("error while cloning payload RawContent: %v", err))
	}
	return newp
}

// MarkAsProcessed cleans the object before putting it back to the pool
func (p *crawlerPayload) MarkAsProcessed() {
	/* 
	   A small optimization trick to reduce the total number of allocations 
	   that are performed while the pipeline is executing. 
	   We set the length of both of the slices and the byte 
	   buffer to zero without modifying their original capacities. 
	   The next time that a recycled payload is sent through the pipeline, 
	   any attempt to write to the byte buffer or append to one of the payload slices will reuse the already allocated space 
	   and only trigger a new memory allocation if additional space is required.
	*/
	p.URL = p.URL[:0]
	p.RawContent.Reset()
	p.NoFollowLinks = p.NoFollowLinks[:0]
	p.Links = p.Links[:0]
	p.Title = p.Title[:0]
	p.TextContent = p.TextContent[:0]
	payloadPool.Put(p)
}

