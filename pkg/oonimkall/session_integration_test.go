package oonimkall_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/enginelocate"
	"github.com/ooni/probe-cli/v3/pkg/oonimkall"
)

func NewSessionForTestingWithAssetsDir(assetsDir string) (*oonimkall.Session, error) {
	return oonimkall.NewSession(&oonimkall.SessionConfig{
		AssetsDir:        assetsDir,
		Logger:           log.Log,
		ProbeServicesURL: "https://api.dev.ooni.io/",
		SoftwareName:     "oonimkall-test",
		SoftwareVersion:  "0.1.0",
		StateDir:         "../testdata/oonimkall/state",
		TempDir:          "../testdata/",
	})
}

func NewSessionForTesting() (*oonimkall.Session, error) {
	return NewSessionForTestingWithAssetsDir("../testdata/oonimkall/assets")
}

func TestNewSessionWithInvalidStateDir(t *testing.T) {
	sess, err := oonimkall.NewSession(&oonimkall.SessionConfig{
		StateDir: "",
	})
	if err == nil || !strings.HasSuffix(err.Error(), "no such file or directory") {
		t.Fatal("not the error we expected")
	}
	if sess != nil {
		t.Fatal("expected a nil Session here")
	}
}

func ReduceErrorForGeolocate(err error) error {
	if err == nil {
		return errors.New("we expected an error here")
	}
	if errors.Is(err, context.Canceled) {
		return nil // when we have not downloaded the resources yet
	}
	if !errors.Is(err, enginelocate.ErrAllIPLookuppersFailed) {
		return nil // otherwise
	}
	return fmt.Errorf("not the error we expected: %w", err)
}

func TestGeolocateWithCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	ctx.Cancel() // cause immediate failure
	location, err := sess.Geolocate(ctx)
	if err := ReduceErrorForGeolocate(err); err != nil {
		t.Fatal(err)
	}
	if location != nil {
		t.Fatal("expected nil location here")
	}
}

func TestGeolocateGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := NewSessionForTesting()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	location, err := sess.Geolocate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if location.ASN == "" {
		t.Fatal("location.ASN is empty")
	}
	if location.Country == "" {
		t.Fatal("location.Country is empty")
	}
	if location.IP == "" {
		t.Fatal("location.IP is empty")
	}
	if location.Org == "" {
		t.Fatal("location.Org is empty")
	}
}

func TestMain(m *testing.M) {
	// Here we're basically testing whether eventually the finalizers
	// will run and the number of active sessions and cancels will become
	// balanced. Especially for the number of active cancels, this is an
	// indication that we've correctly cleaned them up in the session.
	if exitcode := m.Run(); exitcode != 0 {
		os.Exit(exitcode)
	}
	for {
		runtime.GC()
		m, n := oonimkall.ActiveContexts.Load(), oonimkall.ActiveSessions.Load()
		fmt.Printf("./oonimkall: ActiveContexts: %d; ActiveSessions: %d\n", m, n)
		if m == 0 && n == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	os.Exit(0)
}
