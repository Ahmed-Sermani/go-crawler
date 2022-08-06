package frontend

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Ahmed-Sermani/go-crawler/graph"
	"github.com/Ahmed-Sermani/go-crawler/indexer"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

const (
    defaultResultsPageSize = 10
    defaultMaxSummaryLength = 256

    indexEP = "/"
    searchEP = "/search"
    submitLinkEP = "/submit/site"
)

type GraphAPI interface {
    UpsertLink(*graph.Link) error
}

type IndexAPI interface {
    Search(indexer.Query) (indexer.Iterator, error)
}

type Config struct {
    GraphAPI GraphAPI
    IndexAPI IndexAPI
    ListenAddr string

    // The number of results to display per page. If not specified, a default
	// value of 10 results per page will be used instead.
    ResultsPageSize int

    // The maximum length (in characters) of the highlighted content summary for
	// matching documents. If not specified, a default value of 256 will be used
	// instead.
    MaxSummaryLength int

    Logger *logrus.Entry
}

func (cfg *Config) validate() error {
	var err error
	if cfg.ListenAddr == "" {
		err = multierror.Append(err, xerrors.Errorf("listen address has not been specified"))
	}
	if cfg.ResultsPageSize <= 0 {
		cfg.ResultsPageSize = defaultResultsPageSize
	}
	if cfg.MaxSummaryLength <= 0 {
		cfg.MaxSummaryLength = defaultMaxSummaryLength
	}
	if cfg.IndexAPI == nil {
		err = multierror.Append(err, xerrors.Errorf("index API has not been provided"))
	}
	if cfg.GraphAPI == nil {
		err = multierror.Append(err, xerrors.Errorf("graph API has not been provided"))
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.NewEntry(&logrus.Logger{Out: io.Discard})
	}
	return err
}


type Service struct {
    cfg Config
    router *mux.Router
    templateExecutor func(tpl *template.Template, w io.Writer, data map[string]any) error
}


func NewService(cfg Config) (*Service, error) {
	if err := cfg.validate(); err != nil {
		return nil, xerrors.Errorf("front-end service: config validation failed: %w", err)
	}

	svc := &Service{
		router: mux.NewRouter(),
		cfg:    cfg,
		templateExecutor: func(tpl *template.Template, w io.Writer, data map[string]interface{}) error {
			return tpl.Execute(w, data)
		},
	}

	svc.router.HandleFunc(indexEP, svc.renderIndexPage).Methods("GET")
	svc.router.HandleFunc(searchEP, svc.renderSearchResults).Methods("GET")
	svc.router.HandleFunc(submitLinkEP, svc.submitLink).Methods("GET", "POST")
	svc.router.NotFoundHandler = http.HandlerFunc(svc.render404Page)
	return svc, nil
}

// Name implements service.Service
func (svc *Service) Name() string { return "front-end" }

// Run implements service.Service
func (svc *Service) Run(ctx context.Context) error {
	l, err := net.Listen("tcp", svc.cfg.ListenAddr)
	if err != nil {
		return err
	}
	defer func() { _ = l.Close() }()

	srv := &http.Server{
		Addr:    svc.cfg.ListenAddr,
		Handler: svc.router,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	svc.cfg.Logger.WithField("addr", svc.cfg.ListenAddr).Info("starting front-end server")
	if err = srv.Serve(l); err == http.ErrServerClosed {
		// Ignore error when the server shuts down.
		err = nil
	}

	return err
}

func (svc *Service) renderIndexPage(w http.ResponseWriter, _ *http.Request) {
	_ = svc.templateExecutor(indexPageTemplate, w, map[string]any{
		"searchEndpoint":     searchEP,
		"submitLinkEndpoint": submitLinkEP,
	})
}

func (svc *Service) render404Page(w http.ResponseWriter, _ *http.Request) {
	_ = svc.templateExecutor(msgPageTemplate, w, map[string]any{
		"indexEndpoint":  indexEP,
		"searchEndpoint": searchEP,
		"messageTitle":   "Page not found",
		"messageContent": "Page not found.",
	})
}

func (svc *Service) renderSearchErrorPage(w http.ResponseWriter, searchTerms string) {
	w.WriteHeader(http.StatusInternalServerError)
	_ = svc.templateExecutor(msgPageTemplate, w, map[string]interface{}{
		"indexEndpoint":  indexEP,
		"searchEndpoint": searchEP,
		"searchTerms":    searchTerms,
		"messageTitle":   "Error",
		"messageContent": "An error occurred; please try again later.",
	})
}

func (svc *Service) submitLink(w http.ResponseWriter, r *http.Request) {
	var msg string
	defer func() {
		_ = svc.templateExecutor(submitLinkPageTemplate, w, map[string]interface{}{
			"indexEndpoint":      indexEP,
			"submitLinkEndpoint": submitLinkEP,
			"messageContent":     msg,
		})
	}()

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			msg = "Invalid web site URL."
			return
		}
		link, err := url.Parse(r.Form.Get("link"))
		if err != nil || (link.Scheme != "http" && link.Scheme != "https") {
			w.WriteHeader(http.StatusBadRequest)
			msg = "Invalid web site URL."
			return
		}

		link.Fragment = ""
		if err = svc.cfg.GraphAPI.UpsertLink(&graph.Link{URL: link.String()}); err != nil {
			svc.cfg.Logger.WithField("err", err).Errorf("could not upsert link into link graph")
			w.WriteHeader(http.StatusInternalServerError)
			msg = "An error occurred while adding web site to our index; please try again later."
			return
		}

		msg = "Web site was successfully submitted!"
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (svc *Service) renderSearchResults(w http.ResponseWriter, r *http.Request) {
	searchTerms := r.URL.Query().Get("q")
	offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 64)

	matchedDocs, pagination, err := svc.runQuery(searchTerms, offset)
	if err != nil {
		svc.cfg.Logger.WithField("err", err).Errorf("search query execution failed")
		svc.renderSearchErrorPage(w, searchTerms)
		return
	}

	// Render results page
	if err := svc.templateExecutor(resultsPageTemplate, w, map[string]interface{}{
		"indexEndpoint":  indexEP,
		"searchEndpoint": searchEP,
		"searchTerms":    searchTerms,
		"pagination":     pagination,
		"results":        matchedDocs,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (svc *Service) runQuery(searchTerms string, offset uint64) ([]matchedDoc, *paginationDetails, error) {
	var query = indexer.Query{Type: indexer.Match, Expr: searchTerms, Offset: offset}
	if strings.HasPrefix(searchTerms, `"`) && strings.HasSuffix(searchTerms, `"`) {
		query.Type = indexer.Phrase
		searchTerms = strings.Trim(searchTerms, `"`)
	}

	resultIt, err := svc.cfg.IndexAPI.Search(query)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resultIt.Close() }()

	// Wrap each result in a matchedDoc shim and generate a short summary which
	// highlights the matching search terms.
	summarizer := newMatchSummarizer(searchTerms, svc.cfg.MaxSummaryLength)
	highlighter := newMatchHighlighter(searchTerms)
	matchedDocs := make([]matchedDoc, 0, svc.cfg.ResultsPageSize)
	for resCount := 0; resultIt.Next() && resCount < svc.cfg.ResultsPageSize; resCount++ {
		doc := resultIt.Document()
		matchedDocs = append(matchedDocs, matchedDoc{
			doc: doc,
			summary: highlighter.Highlight(
				template.HTMLEscapeString(
					summarizer.MatchSummary(doc.Content),
				),
			),
		})
	}

	if err = resultIt.Error(); err != nil {
		return nil, nil, err
	}

	// Setup paginator and generate prev/next links
	pagination := &paginationDetails{
		From:  int(offset + 1),
		To:    int(offset) + len(matchedDocs),
		Total: int(resultIt.TotalCount()),
	}
	if offset > 0 {
		pagination.PrevLink = fmt.Sprintf("%s?q=%s", searchEP, searchTerms)
		if prevOffset := int(offset) - svc.cfg.ResultsPageSize; prevOffset > 0 {
			pagination.PrevLink += fmt.Sprintf("&offset=%d", prevOffset)
		}
	}
	if nextPageOffset := int(offset) + len(matchedDocs); nextPageOffset < pagination.Total {
		pagination.NextLink = fmt.Sprintf("%s?q=%s&offset=%d", searchEP, searchTerms, nextPageOffset)
	}

	return matchedDocs, pagination, nil
}

// paginationDetails encapsulates the details for rendering a paginator component.
type paginationDetails struct {
	From     int
	To       int
	Total    int
	PrevLink string
	NextLink string
}

// mathcedDoc wraps an index.Document and provides convenience methods for
// rendering its contents in a search results view.
type matchedDoc struct {
	doc     *indexer.Document
	summary string
}

func (d *matchedDoc) HighlightedSummary() template.HTML { return template.HTML(d.summary) }
func (d *matchedDoc) URL() string                       { return d.doc.URL }
func (d *matchedDoc) Title() string {
	if d.doc.Title != "" {
		return d.doc.Title
	}
	return d.doc.URL
}


