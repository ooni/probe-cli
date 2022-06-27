package tlsmiddlebox

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestMeasureTCP(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := NewExperimentMeasurer(Config{})
		ctx := context.Background()
		tcpEvents := make(chan *model.ArchivalTCPConnectResult, 1)
		expected := model.ArchivalTCPConnectResult{
			IP:   "1.1.1.1",
			Port: 80,
			Status: model.ArchivalTCPConnectStatus{
				Failure: nil,
				Success: true,
			},
		}
		err := m.MeasureTCP(ctx, "1.1.1.1:80", tcpEvents)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		out := GetTCPEvents(tcpEvents)
		if len(out) != 1 {
			t.Fatal("expected 1 result, got", len(out))
		}
		if diff := cmp.Diff(*out[0], expected); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("with full channel", func(t *testing.T) {
		m := NewExperimentMeasurer(Config{})
		ctx := context.Background()
		tcpEvents := make(chan *model.ArchivalTCPConnectResult) // channel without buffer
		err := m.MeasureTCP(ctx, "1.1.1.1:80", tcpEvents)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		out := GetTCPEvents(tcpEvents)
		if len(out) > 0 {
			t.Fatal("expected 0 results, got", len(out))
		}
	})
}

func TestTCPConnect_failure(t *testing.T) {
	t.Run("with TCP failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		m := NewExperimentMeasurer(Config{})
		tcpEvents := make(chan *model.ArchivalTCPConnectResult, 1)
		err := m.MeasureTCP(ctx, "1.1.1.1:80", tcpEvents)
		out := GetTCPEvents(tcpEvents)
		expectedFailure := netxlite.FailureInterrupted
		expected := model.ArchivalTCPConnectResult{
			IP:   "1.1.1.1",
			Port: 80,
			Status: model.ArchivalTCPConnectStatus{
				Failure: &expectedFailure,
				Success: false,
			},
		}
		if err == nil {
			t.Fatal("expected error:", netxlite.FailureInterrupted)
		}
		if err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected error", err)
		}
		if len(out) != 1 {
			t.Fatal("expected 1 output, got", len(out))
		}
		if diff := cmp.Diff(*out[0], expected); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("with invalid input", func(t *testing.T) {
		t.Run("with missing port", func(t *testing.T) {
			ctx := context.Background()
			m := NewExperimentMeasurer(Config{})
			tcpEvents := make(chan *model.ArchivalTCPConnectResult, 1)
			err := m.MeasureTCP(ctx, "1.1.1.1", tcpEvents)
			out := GetTCPEvents(tcpEvents)
			// we use a string comparison here since we do not generate OONI failures
			expectedFailure := "address 1.1.1.1: missing port in address"
			if err == nil {
				t.Fatal("expected err")
			}
			if err.Error() != expectedFailure {
				t.Fatal("unexpected error", err)
			}
			if len(out) != 1 {
				t.Fatal("expected 1 output, got", len(out))
			}
			if *(out[0].Status.Failure) != expectedFailure {
				t.Fatal("unexpected failure in output", *(out[0].Status.Failure))
			}
		})
		t.Run("with invalid address", func(t *testing.T) {
			ctx := context.Background()
			m := NewExperimentMeasurer(Config{})
			tcpEvents := make(chan *model.ArchivalTCPConnectResult, 1)
			err := m.MeasureTCP(ctx, "1.1.1.1.2:80", tcpEvents)
			out := GetTCPEvents(tcpEvents)
			// we use a string here since we do not generate OONI failures
			expectedFailure := "no configured resolver"
			if err == nil {
				t.Fatal("expected err")
			}
			if err.Error() != expectedFailure {
				t.Fatal("unexpected error", err)
			}
			if len(out) != 1 {
				t.Fatal("expected 1 output, got", len(out))
			}
			if *(out[0].Status.Failure) != expectedFailure {
				t.Fatal("unexpected failure in output", *(out[0].Status.Failure))
			}
		})
	})
}

func TestWriteToTCPArchival(t *testing.T) {
	var (
		FailureResolver = "no configured resolver"
	)
	type arg struct {
		input string
		err   error
	}
	tests := []struct {
		name    string
		args    arg
		failure string
		want    model.ArchivalTCPConnectResult
	}{{
		name: "with valid input",
		args: arg{
			input: "1.1.1.1:80",
			err:   nil,
		},
		failure: "",
		want: model.ArchivalTCPConnectResult{
			IP:   "1.1.1.1",
			Port: 80,
			Status: model.ArchivalTCPConnectStatus{
				Failure: nil,
				Success: true,
			},
		},
	}, {
		name: "with valid IPv6 input",
		args: arg{
			input: "[2606:2800:220:1:248:1893:25c8:1946]:80",
			err:   nil,
		},
		failure: "",
		want: model.ArchivalTCPConnectResult{
			IP:   "2606:2800:220:1:248:1893:25c8:1946",
			Port: 80,
			Status: model.ArchivalTCPConnectStatus{
				Failure: nil,
				Success: true,
			},
		},
	}, {
		name: "with TCP error",
		args: arg{
			input: "1.1.1.1.1:80",
			err:   errors.New(FailureResolver),
		},
		failure: "no configured resolver",
		want: model.ArchivalTCPConnectResult{
			IP:   "1.1.1.1.1",
			Port: 80,
			Status: model.ArchivalTCPConnectStatus{
				Failure: nil,
				Success: false,
			},
		},
	}, {
		name: "with invalid port",
		args: arg{
			input: "1.1.1.1:port",
			err:   nil,
		},
		failure: `strconv.Atoi: parsing "port": invalid syntax`,
		want: model.ArchivalTCPConnectResult{
			Status: model.ArchivalTCPConnectStatus{
				Failure: nil,
				Success: false,
			},
		},
	}, {
		name: "with invalid host-port combination",
		args: arg{
			input: "1.1.1.1::80",
			err:   nil,
		},
		failure: "address 1.1.1.1::80: too many colons in address",
		want: model.ArchivalTCPConnectResult{
			Status: model.ArchivalTCPConnectStatus{
				Failure: nil,
				Success: false,
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := writeTCPtoArchival(tt.args.input, tt.args.err)
			// We check the Failure filed first since it stores a string pointer
			// this also reflects that an ErrWrapper provides us with a good approach
			// to populate measurements
			if out.Status.Failure != nil {
				if *(out.Status.Failure) != tt.failure {
					t.Fatal("unexpected error", *(out.Status.Failure))
				}
			}
			out.Status.Failure = nil // since we already checked the Failure field
			if diff := cmp.Diff(*out, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
