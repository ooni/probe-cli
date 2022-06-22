package measurexlite

import (
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/fakefill"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewTrace(t *testing.T) {
	const index = 17
	zeroTime := time.Now()
	trace := NewTrace(index, zeroTime)

	t.Run("Index", func(t *testing.T) {
		if trace.Index != index {
			t.Fatal("invalid index")
		}
	})

	t.Run("NetworkEvent has the expected buffer size", func(t *testing.T) {
		ff := &fakefill.Filler{}
		var idx int
	Loop:
		for {
			ev := &model.ArchivalNetworkEvent{}
			ff.Fill(ev)
			select {
			case trace.NetworkEvent <- ev:
				idx++
			default:
				break Loop
			}
		}
		if idx != NetworkEventBufferSize {
			t.Fatal("invalid NetworkEvent channel buffer size")
		}
	})

	t.Run("TCPConnect has the expected buffer size", func(t *testing.T) {
		ff := &fakefill.Filler{}
		var idx int
	Loop:
		for {
			ev := &model.ArchivalTCPConnectResult{}
			ff.Fill(ev)
			select {
			case trace.TCPConnect <- ev:
				idx++
			default:
				break Loop
			}
		}
		if idx != TCPConnectBufferSize {
			t.Fatal("invalid TCPConnect channel buffer size")
		}
	})

	t.Run("TLSHandshake has the expected buffer size", func(t *testing.T) {
		ff := &fakefill.Filler{}
		var idx int
	Loop:
		for {
			ev := &model.ArchivalTLSOrQUICHandshakeResult{}
			ff.Fill(ev)
			select {
			case trace.TLSHandshake <- ev:
				idx++
			default:
				break Loop
			}
		}
		if idx != TLSHandshakeBufferSize {
			t.Fatal("invalid TLSHandshake channel buffer size")
		}
	})

	t.Run("ZeroTime", func(t *testing.T) {
		if !trace.ZeroTime.Equal(zeroTime) {
			t.Fatal("invalid zero time")
		}
	})

	t.Run("dependencies", func(t *testing.T) {
		if trace.dependencies != nil {
			t.Fatal("invalid dependencies")
		}
	})

	t.Run("timeTracker", func(t *testing.T) {
		if trace.timeTracker != nil {
			t.Fatal("invalid time tracker")
		}
	})
}
