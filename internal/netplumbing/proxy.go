package netplumbing

import (
	"net/http"
	"net/url"
)

// httpProxy checks whether we need to use a proxy.
func (txp *Transport) httpProxy(req *http.Request) (*url.URL, error) {
	ctx := req.Context()
	if settings := ContextSettings(ctx); settings != nil && settings.Proxy != nil {
		log := txp.logger(ctx)
		log.Debugf("http: using proxy: %s", settings.Proxy)
		return settings.Proxy, nil
	}
	return nil, nil
}
