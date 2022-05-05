package cdb

import (
	"database/sql"
	"os"
	"testing"

	"github.com/Ahmed-Sermani/search/graph/graphtest"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(CockroachDBGraphTestSuite))

type CockroachDBGraphTestSuite struct {
	graphtest.SuiteBase
	db *sql.DB
}

func Test(t *testing.T) {
	gc.TestingT(t)
} 

func (s *CockroachDBGraphTestSuite) SetUpSuite(c *gc.C) {
	dsn := os.Getenv("CDB_DSN")
	if dsn == "" {
		c.Skip("missing cdb dsn; skipping cdb test package")	
	}

	g, err := NewCockroachDBGraph(dsn)
	c.Assert(err, gc.IsNil)
	s.SetGraph(g)
	s.db = g.db
}

func (s *CockroachDBGraphTestSuite) TearDownSuite(c *gc.C) {
	if s.db != nil {
		s.flushDB(c)
		c.Assert(s.db.Close(), gc.IsNil)
	}
}

func (s *CockroachDBGraphTestSuite) SetUpTest(c *gc.C) {
	s.flushDB(c)
}


func (s *CockroachDBGraphTestSuite) flushDB(c *gc.C) {
	_, err := s.db.Exec("DELETE FROM links")
	c.Assert(err, gc.IsNil)
	_, err = s.db.Exec("DELETE FROM edges")
	c.Assert(err, gc.IsNil)
}
