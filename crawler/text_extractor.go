package crawler

import (
	"context"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
	"github.com/microcosm-cc/bluemonday"
)

var (
	titleRegex                            = regexp.MustCompile(`(?i)<title.*?>(.*?)</title>`)
	repeatedSpaceRegex                    = regexp.MustCompile(`\s+`)
	_                  pipeline.Processor = (*textExtractor)(nil)
)

type textExtractor struct {
	policyPool sync.Pool
}

func newTextExtractor() *textExtractor {
	return &textExtractor{
		policyPool: sync.Pool{
			New: func() any {
				return bluemonday.StrictPolicy()
			},
		},
	}
}

func (te *textExtractor) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	policy := te.policyPool.Get().(*bluemonday.Policy)

	if titleMatch := titleRegex.FindStringSubmatch(payload.RawContent.String()); len(titleMatch) == 2 {
		payload.Title = strings.TrimSpace(html.UnescapeString(repeatedSpaceRegex.ReplaceAllString(
			policy.Sanitize(titleMatch[1]), " ",
		)))
	}

	payload.TextContent = strings.TrimSpace(html.UnescapeString(repeatedSpaceRegex.ReplaceAllString(
		policy.SanitizeReader(&payload.RawContent).String(), " ",
	)))

	te.policyPool.Put(policy)
	return payload, nil
}
