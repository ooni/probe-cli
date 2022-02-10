// The download package contains an experimental experiment aimed
// at helping to investigate heavy throttling.
//
// We are just experimenting for now, so there's not a spec.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "download"
	testVersion = "0.1.0"
)

// Config contains the experiment config.
type Config struct {
	// DNSCache contains a DNS cache.
	DNSCache string `ooni:"Add 'DOMAIN IP...' to cache"`
}

// SpeedSample is a download speed sample.
type SpeedSample struct {
	// T is the elapsed time.
	T float64 `json:"t"`

	// Count is the number of received bytes.
	Count int64 `json:"count"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// ArchivalMeasurement contains all the measurements we performed
	*measurex.ArchivalMeasurement

	// SpeedSamples contains speed samples.
	SpeedSamples []*SpeedSample `json:"speed_samples"`

	// Failure is nil on success and a pointer to string on error
	Failure *string `json:"failure"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// ErrFailure is the error returned when you set the
// config.ReturnError field to true.
var ErrFailure = errors.New("mocked error")

// experimentTimeout is the whole experiment timeout.
const experimentTimeout = 90 * time.Second

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	ctx, cancel := context.WithTimeout(ctx, experimentTimeout)
	defer cancel()
	req, err := measurex.NewHTTPGetRequest(ctx, string(measurement.Input))
	if err != nil {
		return err
	}
	begin := measurement.MeasurementStartTimeSaved
	logger := sess.Logger()
	db := &measurex.MeasurementDB{}
	txp := m.newTransport(begin, logger, db)
	clnt := &http.Client{
		Transport: txp,
		Jar:       measurex.NewCookieJar(),
	}
	resp, err := clnt.Do(req)
	if err != nil {
		measurement.TestKeys = &TestKeys{
			ArchivalMeasurement: measurex.NewArchivalMeasurement(db.AsMeasurement()),
			SpeedSamples:        nil,
			Failure:             m.asFailure(err),
		}
		return nil // this is a valid measurement to submit
	}
	body := newBodyWrapper(callbacks, resp.Body)
	defer body.Close()
	_, err = netxlite.CopyContext(ctx, io.Discard, body)
	if errors.Is(err, context.DeadlineExceeded) {
		err = nil // we've reached the end of the download
	}
	testkeys := &TestKeys{
		ArchivalMeasurement: measurex.NewArchivalMeasurement(db.AsMeasurement()),
		SpeedSamples:        body.moveSamplesOut(),
		Failure:             m.asFailure(err),
	}
	measurement.TestKeys = testkeys
	return nil
}

// newTransport creates a new transport with a default settings except that we're
// not going to trace every I/O event because this will lead to a huge array.
func (m *Measurer) newTransport(begin time.Time,
	logger model.Logger, db *measurex.MeasurementDB) model.HTTPTransport {
	resolver := netxlite.NewResolverStdlib(logger)
	resolver = m.addDNSCache(resolver)
	resolver = measurex.WrapResolver(begin, db, resolver)
	dialer := measurex.WrapDialerWithoutConnWrapping(begin, db,
		netxlite.NewDialerWithoutResolver(logger))
	dialer = netxlite.WrapDialer(logger, resolver, dialer)
	th := measurex.WrapTLSHandshaker(begin, db, netxlite.NewTLSHandshakerStdlib(logger))
	tlsDialer := netxlite.NewTLSDialer(dialer, th)
	const smallBodySnapshot = 1 << 8
	return measurex.WrapHTTPTransport(begin, db,
		netxlite.NewHTTPTransport(logger, dialer, tlsDialer),
		smallBodySnapshot)
}

// addDNSCache wraps an existing resolver to add DNS caching.
func (m *Measurer) addDNSCache(reso model.Resolver) model.Resolver {
	if len(m.config.DNSCache) <= 0 {
		return reso
	}
	cache := make(map[string][]string)
	v := strings.Split(m.config.DNSCache, " ")
	if len(v) >= 2 {
		cache[v[0]] = v[1:]
	}
	return &resolver.CacheResolver{
		Cache:    cache,
		ReadOnly: true,
		Resolver: reso,
	}
}

// bodyWrapper allows to print the download speed and to collect
// download speed samples while we're downloading data.
type bodyWrapper struct {
	// begin is when we wrapped the body.
	begin time.Time

	// callbacks contains the experiment callbacks.
	callbacks model.ExperimentCallbacks

	// cancel allows to cancel the process that logs progress.
	cancel context.CancelFunc

	// count is the number of bytes we've read.
	count *atomicx.Int64

	// mu protects the samples slice.
	mu sync.Mutex

	// ReadCloser is the real underlying body.
	io.ReadCloser

	// samples contains the speed samples.
	samples []*SpeedSample
}

func newBodyWrapper(callbacks model.ExperimentCallbacks, rc io.ReadCloser) *bodyWrapper {
	ctx, cancel := context.WithCancel(context.Background())
	bw := &bodyWrapper{
		begin:      time.Now(),
		callbacks:  callbacks,
		cancel:     cancel,
		count:      &atomicx.Int64{},
		ReadCloser: rc,
	}
	go bw.loop(ctx)
	return bw
}

func (bw *bodyWrapper) moveSamplesOut() (out []*SpeedSample) {
	bw.collectSample(time.Now())
	bw.mu.Lock()
	out = bw.samples
	bw.samples = nil
	bw.mu.Unlock()
	return
}

func (bw *bodyWrapper) loop(ctx context.Context) {
	tkr := time.NewTicker(250 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			bw.collectSample(time.Now())
			return
		case now := <-tkr.C:
			bw.collectSample(now)
		}
	}
}

func (bw *bodyWrapper) collectSample(now time.Time) {
	d := now.Sub(bw.begin)
	total := bw.count.Load()
	elapsed := d.Seconds()
	v := float64(total*8) / elapsed
	bw.mu.Lock()
	bw.samples = append(bw.samples, &SpeedSample{
		T:     d.Seconds(),
		Count: total,
	})
	bw.mu.Unlock()
	uv := humanize.SI(v, "bit/s")
	msg := fmt.Sprintf("average download speed: %s", uv)
	percentage := elapsed / experimentTimeout.Seconds()
	bw.callbacks.OnProgress(percentage, msg)
}

func (bw *bodyWrapper) Read(data []byte) (int, error) {
	count, err := bw.ReadCloser.Read(data)
	if err != nil {
		return 0, err
	}
	bw.count.Add(int64(count))
	return count, nil
}

func (bw *bodyWrapper) Close() error {
	bw.cancel()
	return bw.ReadCloser.Close()
}

// asFailure converts an error to a failure.
func (m *Measurer) asFailure(err error) (out *string) {
	if err != nil {
		s := err.Error()
		out = &s
	}
	return
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
