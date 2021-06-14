// Package hirl contains the HTTP Invalid Request Line network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-007-http-invalid-request-line.md
package hirl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

const (
	testName    = "http_invalid_request_line"
	testVersion = "0.2.0"
	timeout     = 5 * time.Second
)

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment test keys.
type TestKeys struct {
	FailureList   []*string                   `json:"failure_list"`
	Received      []archival.MaybeBinaryValue `json:"received"`
	Sent          []string                    `json:"sent"`
	TamperingList []bool                      `json:"tampering_list"`
	Tampering     bool                        `json:"tampering"`
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{
		Config: config,
		Methods: []Method{
			randomInvalidMethod{},
			randomInvalidFieldCount{},
			randomBigRequestMethod{},
			randomInvalidVersionNumber{},
			squidCacheManager{},
		},
	}
}

// Measurer performs the measurement.
type Measurer struct {
	Config  Config
	Methods []Method
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// ErrNoAvailableTestHelpers is emitted when there are no available test helpers.
	ErrNoAvailableTestHelpers = errors.New("no available helpers")

	// ErrInvalidHelperType is emitted when the helper type is invalid.
	ErrInvalidHelperType = errors.New("invalid helper type")

	// ErrNoMeasurementMethod is emitted when Measurer.Methods is empty.
	ErrNoMeasurementMethod = errors.New("no configured measurement method")
)

// Run implements ExperimentMeasurer.Run.
func (m Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	tk := new(TestKeys)
	measurement.TestKeys = tk
	if len(m.Methods) < 1 {
		return ErrNoMeasurementMethod
	}
	const helperName = "tcp-echo"
	helpers, ok := sess.GetTestHelpersByName(helperName)
	if !ok || len(helpers) < 1 {
		return ErrNoAvailableTestHelpers
	}
	helper := helpers[0]
	if helper.Type != "legacy" {
		return ErrInvalidHelperType
	}
	measurement.TestHelpers = map[string]interface{}{
		"backend": helper.Address,
	}
	out := make(chan MethodResult)
	for _, method := range m.Methods {
		callbacks.OnProgress(0.0, fmt.Sprintf("%s...", method.Name()))
		go method.Run(ctx, MethodConfig{
			Address: helper.Address,
			Logger:  sess.Logger(),
			Out:     out,
		})
	}
	var (
		completed int
		progress  float64
		result    MethodResult
	)
	for {
		select {
		case result = <-out:
		case <-time.After(500 * time.Millisecond):
			if completed <= 0 {
				progress += 0.05
				callbacks.OnProgress(progress, "waiting for results...")
			}
			continue
		}
		failure := archival.NewFailure(result.Err)
		tk.FailureList = append(tk.FailureList, failure)
		tk.Received = append(tk.Received, result.Received)
		tk.Sent = append(tk.Sent, result.Sent)
		tk.TamperingList = append(tk.TamperingList, result.Tampering)
		tk.Tampering = (tk.Tampering || result.Tampering)
		completed++
		percentage := (float64(completed)/float64(len(m.Methods)))*0.5 + 0.5
		callbacks.OnProgress(percentage, fmt.Sprintf("%s... %+v", result.Name, result.Err))
		if completed >= len(m.Methods) {
			break
		}
	}
	return nil
}

// MethodConfig contains the settings for a specific measuring method.
type MethodConfig struct {
	Address string
	Logger  model.Logger
	Out     chan<- MethodResult
}

// MethodResult is the result of one of the methods implemented by this experiment.
type MethodResult struct {
	Err       error
	Name      string
	Received  archival.MaybeBinaryValue
	Sent      string
	Tampering bool
}

// Method is one of the methods implemented by this experiment.
type Method interface {
	Name() string
	Run(ctx context.Context, config MethodConfig)
}

type randomInvalidMethod struct{}

func (randomInvalidMethod) Name() string {
	return "random_invalid_method"
}

func (meth randomInvalidMethod) Run(ctx context.Context, config MethodConfig) {
	RunMethod(ctx, RunMethodConfig{
		MethodConfig: config,
		Name:         meth.Name(),
		RequestLine:  randx.LettersUppercase(4) + " / HTTP/1.1\n\r",
	})
}

type randomInvalidFieldCount struct{}

func (randomInvalidFieldCount) Name() string {
	return "random_invalid_field_count"
}

func (meth randomInvalidFieldCount) Run(ctx context.Context, config MethodConfig) {
	RunMethod(ctx, RunMethodConfig{
		MethodConfig: config,
		Name:         meth.Name(),
		RequestLine: strings.Join([]string{
			randx.LettersUppercase(5),
			" ",
			randx.LettersUppercase(5),
			" ",
			randx.LettersUppercase(5),
			" ",
			randx.LettersUppercase(5),
			"\r\n",
		}, ""),
	})
}

type randomBigRequestMethod struct{}

func (randomBigRequestMethod) Name() string {
	return "random_big_request_method"
}

func (meth randomBigRequestMethod) Run(ctx context.Context, config MethodConfig) {
	RunMethod(ctx, RunMethodConfig{
		MethodConfig: config,
		Name:         meth.Name(),
		RequestLine: strings.Join([]string{
			randx.LettersUppercase(1024),
			" / HTTP/1.1\r\n",
		}, ""),
	})
}

type randomInvalidVersionNumber struct{}

func (randomInvalidVersionNumber) Name() string {
	return "random_invalid_version_number"
}

func (meth randomInvalidVersionNumber) Run(ctx context.Context, config MethodConfig) {
	RunMethod(ctx, RunMethodConfig{
		MethodConfig: config,
		Name:         meth.Name(),
		RequestLine: strings.Join([]string{
			"GET / HTTP/",
			randx.LettersUppercase(3),
			"\r\n",
		}, ""),
	})
}

type squidCacheManager struct{}

func (squidCacheManager) Name() string {
	return "squid_cache_manager"
}

func (meth squidCacheManager) Run(ctx context.Context, config MethodConfig) {
	RunMethod(ctx, RunMethodConfig{
		MethodConfig: config,
		Name:         meth.Name(),
		RequestLine:  "GET cache_object://localhost/ HTTP/1.0\n\r",
	})
}

// RunMethodConfig contains the config for RunMethod
type RunMethodConfig struct {
	MethodConfig
	Name        string
	NewDialer   func(config netx.Config) netx.Dialer
	RequestLine string
}

// RunMethod runs the specific method using the given config and context
func RunMethod(ctx context.Context, config RunMethodConfig) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	result := MethodResult{Name: config.Name}
	defer func() {
		config.Out <- result
	}()
	if config.NewDialer == nil {
		config.NewDialer = netx.NewDialer
	}
	dialer := config.NewDialer(netx.Config{
		ContextByteCounting: true,
		Logger:              config.Logger,
	})
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(config.Address, "80"))
	if err != nil {
		result.Err = err
		return
	}
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		result.Err = err
		return
	}
	if _, err := conn.Write([]byte(config.RequestLine)); err != nil {
		result.Err = err
		return
	}
	result.Sent = config.RequestLine
	data := make([]byte, 4096)
	defer func() {
		result.Tampering = (result.Sent != result.Received.Value)
	}()
	for {
		count, err := conn.Read(data)
		if err != nil {
			// We expect this method to terminate w/ timeout
			if *archival.NewFailure(err) == errorx.FailureGenericTimeoutError {
				err = nil
			}
			result.Err = err
			return
		}
		result.Received.Value += string(data[:count])
	}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	sk.IsAnomaly = tk.Tampering
	return sk, nil
}
