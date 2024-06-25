package wireguard

import (
	"testing"
	"time"

	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewExperimentMeasurer(t *testing.T) {
	m := NewExperimentMeasurer()
	if m.ExperimentName() != "wireguard" {
		t.Fatal("invalid ExperimentName")
	}
	if m.ExperimentVersion() != "0.1.2" {
		t.Fatal("invalid ExperimentVersion")
	}
}

func TestNewEvent(t *testing.T) {
	e := newEvent("foo")
	if e.EventType != "foo" {
		t.Fatal("expected type foo")
	}

	e1 := newEvent("bar")
	e2 := newEvent("baaz")

	log := newEventLogger()
	log.append(e)
	log.append(e1)
	log.append(e2)

	if diff := cmp.Diff(log.log(), []*Event{e, e1, e2}); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewWireguardLogger(t *testing.T) {
	wgLogger := func(events *eventLogger, t int) *device.Logger {
		wgLogger := newWireguardLogger(
			model.DiscardLogger,
			events,
			false,
			time.Now(),
			func(time.Time) time.Duration {
				return time.Duration(t) * time.Second
			})
		return wgLogger
	}

	t.Run("keepalive packet", func(t *testing.T) {
		eventLogger := newEventLogger()
		logger := wgLogger(eventLogger, 2)
		logger.Verbosef(LOG_KEEPALIVE)
		evts := eventLogger.log()
		if len(evts) != 1 {
			t.Fatal("expected 1 event")
		}
		if evts[0].EventType != EVT_RECV_KEEPALIVE {
			t.Fatal("expected RECV_KEEPALIVE")
		}
		if evts[0].T != 2.0 {
			t.Fatal("expected T=2")
		}
	})
	t.Run("handshake send packet", func(t *testing.T) {
		eventLogger := newEventLogger()
		logger := wgLogger(eventLogger, 3)
		logger.Verbosef(LOG_SEND_HANDSHAKE)
		evts := eventLogger.log()
		if len(evts) != 1 {
			t.Fatal("expected 1 event")
		}
		if evts[0].EventType != EVT_SEND_HANDSHAKE_INIT {
			t.Fatal("expected SEND_HANDSHAKE_INIT ")
		}
		if evts[0].T != 3.0 {
			t.Fatal("expected T=3")
		}
	})
	t.Run("handshake recv packet", func(t *testing.T) {
		eventLogger := newEventLogger()
		logger := wgLogger(eventLogger, 4)
		logger.Verbosef(LOG_RECV_HANDSHAKE)
		evts := eventLogger.log()
		if len(evts) != 1 {
			t.Fatal("expected 1 event")
		}
		if evts[0].EventType != EVT_RECV_HANDSHAKE_RESP {
			t.Fatal("expected RECV_HADNSHAKE_RESP ")
		}
		if evts[0].T != 4.0 {
			t.Fatal("expected T=4")
		}
	})

}

// TODO(cleanup) ----
/*

func TestSuccess(t *testing.T) {
	m := NewExperimentMeasurer()
	if m.ExperimentName() != "wireguard" {
		t.Fatal("invalid ExperimentName")
	}
	if m.ExperimentVersion() != "0.1.1" {
		t.Fatal("invalid ExperimentVersion")
	}
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFailure(t *testing.T) {
	m := NewExperimentMeasurer()
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: new(model.Measurement),
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, example.ErrFailure) {
		t.Fatal("expected an error here")
	}
}
*/
