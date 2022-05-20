package es

import (
	"os"
	"strings"
	"testing"

	"github.com/Ahmed-Sermani/search/indexer/indexertest"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(ESIndexerTestSuite))

func Test(t *testing.T) {
	gc.TestingT(t)
}

type ESIndexerTestSuite struct {
	indexertest.SuiteBase
	idx *ESIndexer
}

func (s *ESIndexerTestSuite) SetUpSuite(c *gc.C) {
	nodesList := os.Getenv("ES_NODES")
	if nodesList == "" {
		c.Skip("Missing ES_NODES env; skipping es-based indexer tests")
	}
	indexer, err := NewESIndexer(strings.Split(nodesList, ","))
	c.Assert(err, gc.NotNil)
	s.SetIndexer(indexer)
}

func (s *ESIndexerTestSuite) SetUpTest(c *gc.C) {
	if s.idx.es != nil {
		_, err := s.idx.es.Indices.Delete([]string{indexName})
		c.Assert(err, gc.IsNil)
	}
}
