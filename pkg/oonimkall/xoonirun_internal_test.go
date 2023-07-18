package oonimkall

import "net/url"

// OONIRunFetchWithURL is exposed to tests to exercise ooniRunFetchWithURLLocked
func (sess *Session) OONIRunFetchWithURL(ctx *Context, URL *url.URL) (string, error) {
	sess.mtx.Lock()
	defer sess.mtx.Unlock()
	return sess.ooniRunFetchWithURLLocked(ctx, URL)
}
