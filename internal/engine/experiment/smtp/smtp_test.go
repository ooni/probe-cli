package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func plaintextListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		}
	}
	return l
}

func tlsListener(l net.Listener) net.Listener {
	return tls.NewListener(l, &tls.Config{})
}

func listenerAddr(l net.Listener) string {
	return l.Addr().String()
}

func ValidSMTPServer(conn net.Conn) {
	for {
		command, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return
		}

		if command == "" {
		} else if command == "NOOP" {
			conn.Write([]byte("250 2.0.0 Ok\n"))
		} else if command == "STARTTLS" {
			conn.Write([]byte("220 2.0.0 Ready to start TLS\n"))
			// TODO: conn.Close does not actually close connection? or does client not detect it?
			conn.Close()
			return
		} else if strings.HasPrefix(command, "EHLO") {
			conn.Write([]byte("250 mock.example.com\n"))
		}
		conn.Write([]byte("\n"))
	}
}

func TCPServer(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		defer conn.Close()
		conn.Write([]byte("220 mock.example.com ESMTP (spam is not appreciated)\n"))
		ValidSMTPServer(conn)
	}
}

func TestMeasurer_run(t *testing.T) {
	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{})
		if m.ExperimentName() != "smtp" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.0.1" {
			t.Fatal("invalid experiment version")
		}
		ctx := context.Background()
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}

		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session:     sess,
		}
		err := m.Run(ctx, args)
		return meas, m, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper("")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper("\t")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper("https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with broken TLS", func(t *testing.T) {
		p := plaintextListener()
		defer p.Close()

		l := tlsListener(p)
		defer l.Close()
		addr := listenerAddr(l)
		go TCPServer(l)

		meas, m, err := runHelper("smtps://" + addr)
		if err != nil {
			t.Fatal(err)
		}

		tk := meas.TestKeys.(*TestKeys)

		for _, run := range tk.Runs {
			if *run.TLSHandshake.Failure != "unknown_failure: remote error: tls: unrecognized name" {
				t.Fatal("expected unrecognized_name in TLS handshake")
			}

			if run.noopCounter != 0 {
				t.Fatalf("expected to not have any noops, not %d noops", run.noopCounter)
			}
		}

		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}
	})

	t.Run("with broken starttls", func(t *testing.T) {
		l := plaintextListener()
		defer l.Close()
		addr := listenerAddr(l)

		go TCPServer(l)

		meas, m, err := runHelper("smtp://" + addr)
		if err != nil {
			t.Fatal(err)
		}

		tk := meas.TestKeys.(*TestKeys)

		for _, run := range tk.Runs {
			if *run.TLSHandshake.Failure != "generic_timeout_error" {
				t.Fatal("expected timeout in TLS handshake")
			}

			if run.noopCounter != 0 {
				t.Fatalf("expected to not have any noops, not %d noops", run.noopCounter)
			}
		}

		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}
	})
}
