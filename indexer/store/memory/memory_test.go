package memory

import (
	"testing"

	"github.com/Ahmed-Sermani/go-crawler/indexer/indexertest"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(InMemoryIndexerSuite))

type InMemoryIndexerSuite struct {
	indexertest.SuiteBase
	idx *InMemoryIndexer
}

func Test(t *testing.T) {
	gc.TestingT(t)
}

func (s *InMemoryIndexerSuite) SetUpTest(c *gc.C) {
	idx, err := NewInMemoryBleveIndexer()
	c.Assert(err, gc.IsNil)
	s.SetIndexer(idx)
	s.idx = idx
}

func (s *InMemoryIndexerSuite) TearDownTest(c *gc.C) {
	c.Assert(s.idx.Close(), gc.IsNil)
}
