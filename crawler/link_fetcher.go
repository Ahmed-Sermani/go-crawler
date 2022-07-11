package crawler

import (
	"context"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/Ahmed-Sermani/go-crawler/pipeline"
)

var _ pipeline.Processor = (*linkFetcher)(nil)
var exclusionRegex = regexp.MustCompile(`(?i)\.(?:jpg|jpeg|png|gif|ico|css|js)$`)

type linkFetcher struct {
	urlGetter       URLGetter
	privnetDetector PrivateNetworkDetector
}

func newLinkFetcher(urlGetter URLGetter, privnetDetector PrivateNetworkDetector) *linkFetcher {
	return &linkFetcher{
		urlGetter:       urlGetter,
		privnetDetector: privnetDetector,
	}
}

func (lf *linkFetcher) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)
	// Skip URLs that point to files that cannot contain html content.
	if exclusionRegex.MatchString(payload.URL) {
		return nil, nil
	}

	// Skip URLs that point to files that cannot contain html content.
	if exclusionRegex.MatchString(payload.URL) {
		return nil, nil
	}

	// Never crawl links in private networks (e.g. link-local addresses).
	if isPrivate, err := lf.isPrivate(payload.URL); err != nil || isPrivate {
		return nil, nil
	}

	res, err := lf.urlGetter.Get(payload.URL)
	if err != nil {
		return nil, nil
	}
	defer res.Body.Close()

	// Skip payloads for failure status codes.
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, nil
	}

	// Skip payloads for non-html payloads
	if contentType := res.Header.Get("Content-Type"); !strings.Contains(contentType, "html") {
		return nil, nil
	}

	_, err = io.Copy(&payload.RawContent, res.Body)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (lf *linkFetcher) isPrivate(URL string) (bool, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return false, err
	}
	return lf.privnetDetector.IsPrivate(u.Hostname())
}
