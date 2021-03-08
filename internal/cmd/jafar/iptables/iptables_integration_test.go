package iptables

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/execabs"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/resolver"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/uncensored"
	"github.com/ooni/probe-cli/v3/internal/engine/shellx"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}

func newCensoringPolicy() *CensoringPolicy {
	policy := NewCensoringPolicy()
	policy.Waive() // start over to allow for repeated tests on failure
	return policy
}

func TestCannotApplyPolicy(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.DropIPs = []string{"antani"}
	if err := policy.Apply(); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestCreateChainsError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	// you should not be able to apply the policy when there is
	// already a policy, you need to waive it first
	if err := policy.Apply(); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestDropIP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.DropIPs = []string{"1.1.1.1"}
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", "1.1.1.1:853")
	if err == nil {
		t.Fatalf("expected an error here")
	}
	if err.Error() != "dial tcp 1.1.1.1:853: i/o timeout" {
		t.Fatal("unexpected error occurred")
	}
	if conn != nil {
		t.Fatal("expected nil connection here")
	}
}

func TestDropKeyword(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.DropKeywords = []string{"ooni.io"}
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequest("GET", "http://www.ooni.io", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !strings.HasSuffix(err.Error(), "context deadline exceeded") {
		t.Fatal("unexpected error occurred")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestDropKeywordHex(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.DropKeywordsHex = []string{"|6f 6f 6e 69|"}
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequest("GET", "http://www.ooni.io", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err == nil {
		t.Fatal("expected an error here")
	}
	// the error we see with GitHub Actions is different from the error
	// we see when testing locally on Fedora
	if !strings.HasSuffix(err.Error(), "operation not permitted") &&
		!strings.HasSuffix(err.Error(), "Temporary failure in name resolution") &&
		!strings.HasSuffix(err.Error(), "no such host") {
		t.Fatalf("unexpected error occurred: %+v", err)
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestResetIP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.ResetIPs = []string{"1.1.1.1"}
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	conn, err := (&net.Dialer{}).Dial("tcp", "1.1.1.1:853")
	if err == nil {
		t.Fatalf("expected an error here")
	}
	if err.Error() != "dial tcp 1.1.1.1:853: connect: connection refused" {
		t.Fatal("unexpected error occurred")
	}
	if conn != nil {
		t.Fatal("expected nil connection here")
	}
}

func TestResetKeyword(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.ResetKeywords = []string{"ooni.io"}
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get("http://www.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if strings.Contains(err.Error(), "read: connection reset by peer") == false {
		t.Fatal("unexpected error occurred")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestResetKeywordHex(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.ResetKeywordsHex = []string{"|6f 6f 6e 69|"}
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get("http://www.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if strings.Contains(err.Error(), "read: connection reset by peer") == false {
		t.Fatal("unexpected error occurred")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestHijackDNS(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	resolver := resolver.NewCensoringResolver(
		[]string{"ooni.io"}, nil, nil,
		uncensored.Must(uncensored.NewClient("dot://1.1.1.1:853")),
	)
	server, err := resolver.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown()
	policy := newCensoringPolicy()
	defer policy.Waive()
	policy.HijackDNSAddress = server.PacketConn.LocalAddr().String()
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	addrs, err := net.LookupHost("www.ooni.io")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if strings.Contains(err.Error(), "no such host") == false {
		t.Fatal("unexpected error occurred")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
}

func TestHijackHTTP(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// Implementation note: this test is complicated by the fact
	// that we are running as root and so we're whitelisted.
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(451)
		}),
	)
	defer server.Close()
	policy := newCensoringPolicy()
	defer policy.Waive()
	pu, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	policy.HijackHTTPAddress = pu.Host
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	err = shellx.Run("sudo", "-u", "nobody", "--",
		"curl", "-sf", "http://example.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	var exitErr *execabs.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("not the error type we expected")
	}
	if exitErr.ExitCode() != 22 {
		t.Fatal("not the exit code we expected")
	}
}

func TestHijackHTTPS(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("not implemented on this platform")
	}
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// Implementation note: this test is complicated by the fact
	// that we are running as root and so we're whitelisted.
	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(451)
		}),
	)
	defer server.Close()
	policy := newCensoringPolicy()
	defer policy.Waive()
	pu, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	policy.HijackHTTPSAddress = pu.Host
	if err := policy.Apply(); err != nil {
		t.Fatal(err)
	}
	err = shellx.Run("sudo", "-u", "nobody", "--",
		"curl", "-sf", "https://example.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	t.Log(err)
	var exitErr *execabs.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("not the error type we expected")
	}
	if exitErr.ExitCode() != 60 {
		t.Fatal("not the exit code we expected")
	}
}
