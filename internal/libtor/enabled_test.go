//go:build ooni_libtor

package libtor

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cretz/bine/tor"
)

func TestNormalUsage(t *testing.T) {
	ctx := context.Background()

	datadir, err := filepath.Abs(filepath.Join("testdata", "datadir"))
	if err != nil {
		t.Fatal(err)
	}

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}
	config := &tor.StartConf{
		ProcessCreator: creator,
		DataDir:        datadir,
		ExtraArgs:      nil,
		NoHush:         true,
	}

	instance, err := tor.Start(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()

	if err := instance.EnableNetwork(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}

func TestContextAlreadyExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	process, err := creator.New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure that start happens after the background
	// goroutine noticing that ctx is cancelled.
	<-process.(*torProcess).closedWhenNotStarted

	if err := process.Start(); !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
}

func TestTooManyCommandLineArguments(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	argv := make([]string, 4096)
	process, err := creator.New(ctx, argv...)
	if err != nil {
		t.Fatal(err)
	}

	if err := process.Start(); !errors.Is(err, ErrTooManyArguments) {
		t.Fatal("unexpected err", err)
	}
}

func TestSetupControlSocketFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	process, err := creator.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	process.(*torProcess).simulateBadControlSocket = true

	if err := process.Start(); !errors.Is(err, ErrCannotCreateControlSocket) {
		t.Fatal("unexpected err", err)
	}
}

func TestFileConnFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	process, err := creator.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	process.(*torProcess).simulateFileConnFailure = true

	if err := process.Start(); !errors.Is(err, ErrCannotCreateControlSocket) {
		t.Fatal("unexpected err", err)
	}
}

func TestNonzeroExitCode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	process, err := creator.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	process.(*torProcess).simulateNonzeroExitCode = true

	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	if err := process.Wait(); !errors.Is(err, ErrNonzeroExitCode) {
		t.Fatal("unexpected err", err)
	}
}

func TestContextCanceledWhileTorIsRunning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	process, err := creator.New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	cconn, err := process.EmbeddedControlConn()
	if err != nil {
		t.Fatal(err)
	}

	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	message := []byte("SETEVENTS STATUS_CLIENT\r\n")
	if _, err := cconn.Write(message); err != nil {
		t.Fatal(err)
	}

	for {
		message := make([]byte, 1<<20)
		count, err := cconn.Read(message)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		message = message[:count]
		t.Log(strings.Trim(string(message), "\r\n"))
	}

	if err := process.Wait(); err != nil {
		t.Fatal(err)
	}
}

// In theory this case SHOULD NOT happen judging from the description of
// the EmbeddedControlConn method, which reads:
//
//	Note, this should only be called once per process before
//	Start, and the connection does not need to be closed.
//
// However, it MIGHT happen. So, let's also cover this possibility.
func TestControlConnectionExplicitlyClosed(t *testing.T) {
	ctx := context.Background()

	creator, good := MaybeCreator()
	if !good {
		t.Fatal("expected to see true here")
	}

	process, err := creator.New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	cconn, err := process.EmbeddedControlConn()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		<-time.After(2 * time.Second)
		cconn.Close()
	}()

	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	message := []byte("SETEVENTS STATUS_CLIENT\r\n")
	if _, err := cconn.Write(message); err != nil {
		t.Fatal(err)
	}

	for {
		message := make([]byte, 1<<20)
		count, err := cconn.Read(message)
		if errors.Is(err, io.ErrClosedPipe) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		message = message[:count]
		t.Log(strings.Trim(string(message), "\r\n"))
	}

	if err := process.Wait(); err != nil {
		t.Fatal(err)
	}
}
