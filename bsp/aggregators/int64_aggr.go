package aggregators

import (
	"sync/atomic"

	"github.com/Ahmed-Sermani/go-crawler/bsp"
)

var _ bsp.Aggregator = (*IntAggregator)(nil)

// IntAggregator implements a concurrent-safe accumlator for int values.
// It uses mutex free implementation.
type IntAggregator struct {
    prevSum, curSum int64
}

func (a *IntAggregator) Type() string {
    return "IntAggregator"
}

func (a *IntAggregator) Get() any {
    return int(atomic.LoadInt64(&a.curSum))
}

func (a *IntAggregator) Set(v any) {
    for v64 := int64(v.(int)); ; {
        oldCur := a.curSum
        oldPrev := a.prevSum
        if atomic.CompareAndSwapInt64(&a.curSum, oldCur, v64) && atomic.CompareAndSwapInt64(&a.prevSum, oldPrev, v64) {
            return
        }
    }
}

func (a *IntAggregator) Aggregate(v any) {
    _ = atomic.AddInt64(&a.curSum, int64(v.(int)))
}

func (a *IntAggregator) Delta() any {
    for {
        curSum, preSum := atomic.LoadInt64(&a.curSum) , atomic.LoadInt64(&a.prevSum)
        if atomic.CompareAndSwapInt64(&a.prevSum, preSum, curSum) {
            return int(curSum - preSum)
        }
    }
}
