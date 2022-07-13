package aggregators

import (
	"math"
	"math/rand"
	"testing"

	"github.com/Ahmed-Sermani/go-crawler/bsp"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(AggregatorTestSuite))

func Test(t *testing.T) {
	gc.TestingT(t)
}

type AggregatorTestSuite struct{}

func (s *AggregatorTestSuite) TestIntAggregator(c *gc.C) {
	numValues := 100
	values := make([]interface{}, numValues)
	var exp int
	for i := 0; i < numValues; i++ {
		next := rand.Int()
		values[i] = next
		exp += next
	}

	got := s.testConcurrentAccess(new(IntAggregator), values).(int)
	c.Assert(got, gc.Equals, exp)
}

func (s *AggregatorTestSuite) TestFloat64Accumulator(c *gc.C) {
	numValues := 100
	values := make([]interface{}, numValues)
	var exp float64
	for i := 0; i < numValues; i++ {
		next := rand.Float64()
		values[i] = next
		exp += next
	}

	got := s.testConcurrentAccess(new(Float64Aggregator), values).(float64)
	absDelta := math.Abs(exp - got)
	c.Assert(absDelta < 1e-6, gc.Equals, true, gc.Commentf("expected to get %f; got %f; |delta| %f > 1e-6", exp, got, absDelta))
}

func (s *AggregatorTestSuite) testConcurrentAccess(a bsp.Aggregator, values []interface{}) interface{} {
	startedCh := make(chan struct{})
	syncCh := make(chan struct{})
	doneCh := make(chan struct{})
	for i := 0; i < len(values); i++ {
		go func(i int) {
			startedCh <- struct{}{}
			<-syncCh
			a.Aggregate(values[i])
			doneCh <- struct{}{}
		}(i)
	}

	// Wait for all go-routines to start
	for i := 0; i < len(values); i++ {
		<-startedCh
	}

	close(syncCh)

	// Wait for all go-routines to exit
	for i := 0; i < len(values); i++ {
		<-doneCh
	}

	return a.Get()
}
