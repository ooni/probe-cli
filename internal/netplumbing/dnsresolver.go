package netplumbing

import (
	"context"
	"errors"
	"net/url"
	"sync"
)

// NewDNSResolver creates a new Resolver implementation using the
// specified URL and this transport to perform domain name resolutions. You
// typically want to use the object returned by this call to override the
// underlying resolver using the Config.Resolver field.
func (txp *Transport) NewDNSResolver(URL *url.URL) *DNSResolver {
	return &DNSResolver{Transport: txp, URL: URL}
}

// DNSResolver is a DNS resolver using a given resolver URL and transport.
type DNSResolver struct {
	// Transport is the mandatory transport to use.
	Transport *Transport

	// URL is the resolver URL. (See DNSQuery's documentation.)
	URL *url.URL
}

// LookupHost searchers for the A and AAAA domains associated with
// the given domain using the specified resolver URL.
func (r *DNSResolver) LookupHost(ctx context.Context, domain string) ([]string, error) {
	wg := &sync.WaitGroup{}
	workerA := &dnsLookupHostWorker{
		ctx:    ctx,
		url:    r.URL,
		domain: domain,
		name:   "dnsLookupA",
		f:      r.Transport.dnsLookupHostA,
		wg:     wg,
	}
	workerA.do()
	workerAAAA := &dnsLookupHostWorker{
		ctx:    ctx,
		url:    r.URL,
		domain: domain,
		name:   "dnsLookupAAAA",
		f:      r.Transport.dnsLookupHostAAAA,
		wg:     wg,
	}
	workerAAAA.do()
	wg.Wait()
	if workerA.err != nil && workerAAAA.err != nil {
		return nil, workerA.err
	}
	var addrs []string
	addrs = append(addrs, workerA.addrs...)
	addrs = append(addrs, workerAAAA.addrs...)
	if len(addrs) < 1 {
		return nil, errors.New("netplumbing: no IP address returned")
	}
	return addrs, nil
}

// dnsLookupHostWorker is a worker for DNSLookupHost
type dnsLookupHostWorker struct {
	// ctx is the context for this work. Do not modify this
	// field once you've called do.
	ctx context.Context

	// url is the resolver url. Do not modify this
	// field once you've called do.
	url *url.URL

	// domain is the input domain. Do not modify this
	// field once you've called do.
	domain string

	// name is the trace group name to use. Do not modify this
	// field once you've called do.
	name string

	// f is the function to call. Do not modify this
	// field once you've called do.
	f func(ctx context.Context, resolverURL *url.URL,
		domain string) (addrs []string, err error)

	// wg is the related wait group. Do not modify this
	// field once you've called do.
	wg *sync.WaitGroup

	// addrs contains the resulting addresses. It is safe to
	// access this field only after wg.Wait returns.
	addrs []string

	// err contains the resulting error. It is safe to access
	// this field only after wg.Wait returns.
	err error
}

// do runs the worker
func (w *dnsLookupHostWorker) do() {
	w.wg.Add(1)
	go func() {
		w.addrs, w.err = w.maybeTraceF(w.ctx, w.url, w.domain)
		w.wg.Done()
	}()
}

// maybeTraceF decides whether we want to trace f
func (w *dnsLookupHostWorker) maybeTraceF(
	ctx context.Context, resolverURL *url.URL, domain string) ([]string, error) {
	if th := ContextTraceHeader(ctx); th != nil {
		return w.traceF(ctx, resolverURL, domain, th)
	}
	return w.f(ctx, resolverURL, domain)
}

// traceF runs f with tracing.
func (w *dnsLookupHostWorker) traceF(
	ctx context.Context, resolverURL *url.URL, domain string,
	ht *TraceHeader) ([]string, error) {
	child := &TraceHeader{}
	addrs, err := w.f(WithTraceHeader(ctx, child), resolverURL, domain)
	ht.add(&GroupTrace{Children: child.MoveOut(), Name: w.name})
	return addrs, err
}

// dnsLookupHostA searches for A IP addresses.
func (txp *Transport) dnsLookupHostA(
	ctx context.Context, resolverURL *url.URL, domain string) ([]string, error) {
	query := txp.DNSEncodeA(domain, txp.dnsNeedsPadding(resolverURL))
	reply, err := txp.DNSQuery(ctx, resolverURL, query)
	if err != nil {
		return nil, err
	}
	return txp.DNSDecodeA(reply)
}

func (txp *Transport) dnsNeedsPadding(u *url.URL) bool {
	return u.Scheme == "dot" || u.Scheme == "https"
}

// dnsLookupHostAAAA searches for AAAA IP addresses.
func (txp *Transport) dnsLookupHostAAAA(
	ctx context.Context, resolverURL *url.URL, domain string) ([]string, error) {
	query := txp.DNSEncodeAAAA(domain, txp.dnsNeedsPadding(resolverURL))
	reply, err := txp.DNSQuery(ctx, resolverURL, query)
	if err != nil {
		return nil, err
	}
	return txp.DNSDecodeAAAA(reply)
}
