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

	process, err := creator.New(ctx, "SocksPort", "auto")
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

	process, err := creator.New(ctx, "SocksPort", "auto")
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

// This test ensures that we cannot make concurrent calls to the library.
func TestConcurrentCalls(t *testing.T) {
	// we need to simulate non zero exit code here such that we're not
	// actually hitting into the real tor library; by doing this we
	// make the test faster and reduce the risk of triggering the
	// https://github.com/ooni/probe/issues/2406 bug caused by the
	// fact we're invoking tor multiple times.

	run := func(startch chan<- error) {
		ctx := context.Background()

		creator, good := MaybeCreator()
		if !good {
			t.Fatal("expected to see true here")
		}

		process, err := creator.New(ctx)
		if err != nil {
			t.Fatal(err)
		}
		process.(*torProcess).simulateNonzeroExitCode = true // don't actually run tor

		cconn, err := process.EmbeddedControlConn()
		if err != nil {
			t.Fatal(err)
		}
		defer cconn.Close()

		// we expect a process to either start successfully or fail because
		// there are concurrent calls ongoing
		err = process.Start()
		if err != nil && !errors.Is(err, ErrConcurrentCalls) {
			t.Fatal("unexpected err", err)
		}
		t.Log("seen this error coming from process.Start", err)
		startch <- err
		if err != nil {
			return
		}

		// the process that starts should complain about a nonzero
		// exit code because it's configured in this way
		if err := process.Wait(); !errors.Is(err, ErrNonzeroExitCode) {
			t.Fatal("unexpected err", err)
		}
	}

	// attempt to create N=5 parallel instances
	//
	// what we would expect to see is that just one instance
	// is able to start while the other four instances fail instead
	// during their startup phase because of concurrency
	const concurrentRuns = 5
	start := make(chan error, concurrentRuns)
	for idx := 0; idx < concurrentRuns; idx++ {
		go run(start)
	}
	var (
		countGood          int
		countConcurrentErr int
	)
	for idx := 0; idx < concurrentRuns; idx++ {
		err := <-start
		if err == nil {
			countGood++
			continue
		}
		if errors.Is(err, ErrConcurrentCalls) {
			countConcurrentErr++
			continue
		}
		t.Fatal("unexpected error", err)
	}
	if countGood != 1 {
		t.Fatal("expected countGood == 1, got", countGood)
	}
	if countConcurrentErr != 4 {
		t.Fatal("expected countConcurrentErr == 4, got", countConcurrentErr)
	}
}
