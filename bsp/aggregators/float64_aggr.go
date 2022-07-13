package aggregators

import (
	"math"
	"sync/atomic"
	"unsafe"
)

type Float64Aggregator struct {
	curSum, prevSum float64
}

func (a *Float64Aggregator) Type() string {
	return "Float64Aggregator"
}

func (a *Float64Aggregator) Get() any {
	return loadFloat64(&a.curSum)
}

func (a *Float64Aggregator) Set(v any) {
	for v64 := v.(float64); ; {
		oldCur, oldPrev := loadFloat64(&a.curSum), loadFloat64(&a.prevSum)
		swappedC := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.curSum)),
			math.Float64bits(oldCur),
			math.Float64bits(v64),
		)
		swappedP := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.prevSum)),
			math.Float64bits(oldPrev),
			math.Float64bits(v64),
		)
		if swappedC && swappedP {
			return
		}
	}
}

func (a *Float64Aggregator) Aggregate(v any) {
	for v64 := v.(float64); ; {
		oldCur := loadFloat64(&a.curSum)
		newCur := oldCur + v64
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.curSum)),
			math.Float64bits(oldCur),
			math.Float64bits(newCur),
		) {
			return
		}
	}
}

func (a *Float64Aggregator) Delta() any {
	for {
		curSum, prevSum := loadFloat64(&a.curSum), loadFloat64(&a.prevSum)
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.prevSum)),
			math.Float64bits(prevSum),
			math.Float64bits(curSum),
		) {
			return curSum - prevSum
		}
	}
}

// atomic load for float64
// it works by casting float64 to uint64 then load the latter.
func loadFloat64(fp *float64) float64 {
	return math.Float64frombits(
		atomic.LoadUint64((*uint64)(unsafe.Pointer(fp))),
	)
}
