package urlgetter

import "sync/atomic"

// IndexGen generates new trace indexes.
//
// The zero value is ready to use.
type IndexGen struct {
	idx atomic.Int64
}

var _ RunnerTraceIndexGenerator = &IndexGen{}

// Next implements [RunnerTraceIndexGenerator].
func (ig *IndexGen) Next() int64 {
	return ig.idx.Add(1)
}
