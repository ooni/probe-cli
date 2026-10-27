//go:build netns && linux

package tlsmiddlebox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestNetns runs the experiment across the chain of network namespaces that testdata/netns/topo.sh
// builds, with real kernel routers: packets sent with TTL 1-3 expire at routers r1-r3 and TTL 4
// reaches the server, which refuses SNIs it doesn't serve. Each case can put a middlebox rule on
// r2, which sees the packets sent with TTL >= 3. It needs root in the probe's namespace, so it
// only builds with -tags netns: run it with testdata/netns/run.sh (rootless podman), or as CI
// does (.github/workflows/tlsmiddlebox.yml).
func TestNetns(t *testing.T) {
	const (
		ok      = "" // the handshake succeeded
		hu      = "host_unreachable"
		timeout = "generic_timeout_error"
		reset   = "connection_reset"
		alert   = "unknown_failure: remote error: tls: handshake failure"
		match   = "-p tcp -m string --algo bm --string target.test"
	)
	cases := []struct {
		name        string
		rules       []string // for r2's FORWARD chain, in order
		control     string   // the control SNI
		clientID    int64    // the uTLS fingerprint (ClientIDs), or 0 for Go's ClientHello
		wantTarget  []string
		wantControl []string
	}{{
		name:        "no middlebox",
		control:     "control.test",
		wantTarget:  []string{hu, hu, hu, ok},
		wantControl: []string{hu, hu, hu, ok},
	}, {
		name:        "the server refuses the control SNI, as CloudFront refuses example.com",
		control:     "rejected.test",
		wantTarget:  []string{hu, hu, hu, ok},
		wantControl: []string{hu, hu, hu, alert, alert, alert},
	}, {
		// Go's ClientHello (1.5 kB, with the X25519MLKEM768 key share) takes two segments, and
		// only the first has the SNI. With TTL 3 the second passes r2 and expires at r3, whose
		// time exceeded fails the read, so the drop shows from TTL 4: one hop late
		name:        "r2 drops the segment with the target SNI",
		rules:       []string{match + " -j DROP"},
		control:     "control.test",
		wantTarget:  []string{hu, hu, hu, timeout, timeout, timeout},
		wantControl: []string{hu, hu, hu, ok},
	}, {
		// the uTLS fingerprints are old (gitlab.com/yawning/utls): no ML-KEM, one segment
		name:        "r2 drops the target SNI in uTLS Chrome's one-segment ClientHello",
		rules:       []string{match + " -j DROP"},
		control:     "control.test",
		clientID:    2,
		wantTarget:  []string{hu, hu, timeout, timeout, timeout, timeout},
		wantControl: []string{hu, hu, hu, ok},
	}, {
		name:        "r2 drops the flow from the segment with the target SNI",
		rules:       []string{match + " -j CONNMARK --set-mark 1", "-m connmark --mark 1 -j DROP"},
		control:     "control.test",
		wantTarget:  []string{hu, hu, timeout, timeout, timeout, timeout},
		wantControl: []string{hu, hu, hu, ok},
	}, {
		name:        "r2 resets the target SNI",
		rules:       []string{match + " -j REJECT --reject-with tcp-reset"},
		control:     "control.test",
		wantTarget:  []string{hu, hu, reset},
		wantControl: []string{hu, hu, hu, ok},
	}, {
		// On an established connection the kernel reports an ICMP destination unreachable as
		// EHOSTUNREACH, as it does a time exceeded, and queues neither on the socket's error
		// queue: this trace looks like one that never reaches the server. Only comparing it
		// with the traceroute (connected from TTL 4) tells them apart
		name:        "r2 rejects the target SNI with ICMP admin-prohibited",
		rules:       []string{match + " -j REJECT --reject-with icmp-admin-prohibited"},
		control:     "control.test",
		wantTarget:  []string{hu, hu, hu, hu, hu, hu},
		wantControl: []string{hu, hu, hu, ok},
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rule := range tc.rules {
				r2(t, "-A FORWARD "+rule)
				defer r2(t, "-D FORWARD "+rule)
			}
			trace := runNetns(t, tc.control, tc.clientID)
			// time exceeded from r1-r3, then connected: no case filters SYNs
			if got := traceroute(trace.TCPTraceroute); got != "1:11/0 2:11/0 3:11/0 4:C 5:C 6:C" {
				t.Errorf("traceroute: %s", got)
			}
			check(t, "target", trace.TargetTrace, "target.test", tc.wantTarget)
			check(t, "control", trace.ControlTrace, tc.control, tc.wantControl)
		})
	}
}

// r2 runs iptables in r2's namespace.
func r2(t *testing.T, rule string) {
	args := append([]string{"--net=/run/netns/r2", "iptables"}, strings.Fields(rule)...)
	if out, err := exec.Command("nsenter", args...).CombinedOutput(); err != nil {
		t.Fatalf("iptables %s: %s %s", rule, err, out)
	}
}

// runNetns measures https://target.test at topo.sh's server.
func runNetns(t *testing.T, control string, clientID int64) *CompleteTrace {
	meas := &model.Measurement{Input: "https://target.test", MeasurementStartTimeSaved: time.Now()}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
		Measurement: meas,
		Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		Target: &Target{
			URL: "https://target.test",
			Config: &Config{
				TestHelper: "tlshandshake://10.0.4.2:443",
				SNIControl: control,
				ClientId:   clientID,
				MaxTTL:     6,
			},
		},
	}
	if err := NewExperimentMeasurer().Run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	// the probe's own address must not reach the measurement, e.g. through an
	// error string such as "read tcp <local>-><remote>"
	data, err := json.Marshal(meas.TestKeys)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(`10.0.1.2`)) {
		t.Errorf("the test keys contain the probe's address 10.0.1.2: %s", data)
	}
	if !bytes.Contains(data, []byte(`10.0.4.2`)) {
		t.Errorf("the test keys lack the server's address, so the check above proves nothing")
	}
	traces := meas.TestKeys.(*TestKeys).IterativeTrace
	if len(traces) != 1 {
		t.Fatalf("got %d traces, want 1", len(traces))
	}
	return traces[0]
}

// check compares a trace's failures with want, by TTL from 1, and logs each iteration.
func check(t *testing.T, side string, tr *IterativeTrace, sni string, want []string) {
	if tr.SNI != sni {
		t.Errorf("%s: SNI %q, want %q", side, tr.SNI, sni)
	}
	var got []string
	for _, it := range tr.Iterations {
		h := it.Handshake
		failure := str(h.Failure)
		got = append(got, failure)
		t.Logf("%s TTL %d: %-55q so_error=%q address=%q t=%.3f",
			side, it.TTL, failure, str(h.SoError), h.Address, h.T)
		if h.Address != "10.0.4.2:443" || h.T <= 0 {
			t.Errorf("%s TTL %d: the handshake lost its address or time", side, it.TTL)
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("%s: failures by TTL\n got %q\nwant %q", side, got, want)
	}
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// traceroute summarises the TCP traceroute: the ICMP type/code per TTL, or C for connected.
func traceroute(tr *IterativeTraceroute) string {
	var out []string
	for _, it := range tr.Iterations {
		e := it.ICMPError
		switch {
		case e.Connected:
			out = append(out, fmt.Sprintf("%d:C", it.TTL))
		case e.Timeout:
			out = append(out, fmt.Sprintf("%d:timeout", it.TTL))
		default:
			out = append(out, fmt.Sprintf("%d:%d/%d", it.TTL, e.Type, e.Code))
		}
	}
	return strings.Join(out, " ")
}
