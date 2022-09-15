package webconnectivity

import "github.com/ooni/probe-cli/v3/internal/atomicx"

// NumRedirects counts the number of redirects left.
type NumRedirects struct {
	count *atomicx.Int64
}

// NewNumRedirects creates a new NumRedirects instance.
func NewNumRedirects(n int64) *NumRedirects {
	count := &atomicx.Int64{}
	count.Add(n)
	return &NumRedirects{
		count: count,
	}
}

// CanFollowOneMoreRedirect returns true if we are
// allowed to follow one more redirect.
func (nr *NumRedirects) CanFollowOneMoreRedirect() bool {
	return nr.count.Add(-1) > 0
}
