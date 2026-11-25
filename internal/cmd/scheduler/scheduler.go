// scheduler.go
//
// Build: go build -o scheduler scheduler.go
// Usage: INTERVAL=86400 MAX_MULTIPLIER=10 COUNTDOWN=true ./scheduler /path/to/cmd arg1 arg2
package main

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	mathrand "math/rand"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func getenvInt(name string, def int) int {
	if v, ok := os.LookupEnv(name); ok {
		if i, err := strconv.Atoi(v); err == nil && i >= 0 {
			return i
		}
	}
	return def
}

func getenvBool(name string, def bool) bool {
	if v, ok := os.LookupEnv(name); ok {
		if v == "1" || v == "true" || v == "TRUE" {
			return true
		}
		if v == "0" || v == "false" || v == "FALSE" {
			return false
		}
	}
	return def
}

func seedMathRand() {
	var b [8]byte
	if _, err := crand.Read(b[:]); err == nil {
		seed := int64(binary.LittleEndian.Uint64(b[:]))
		mathrand.Seed(seed)
	} else {
		mathrand.Seed(time.Now().UnixNano())
	}
}

func expDelaySeconds(mean float64, maxMultiplier int) (int, error) {
	// drop outliers and sample again
	maxAllowed := mean * float64(maxMultiplier)
	minAllowed := mean * 0.1
	for attempts := 0; attempts < 1_000_000; attempts++ {
		u := mathrand.Float64() // [0,1)
		if u == 0 {
			u = math.SmallestNonzeroFloat64
		}
		d := -mean * math.Log(u) // (-inf, 1]
		if d <= maxAllowed && d >= minAllowed {
			return int(d), nil
		}
	}
	return 0, errors.New("failed to draw delay within allowed range")
}

func runOnce(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "no command supplied")
		return 2
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start error: %v\n", err)
		return 1
	}
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			return ws.ExitStatus()
		}
	}
	fmt.Fprintf(os.Stderr, "process error: %v\n", err)
	return 1
}

// print countdown; return true if cancelled
func countdownLoop(ctx context.Context, seconds int) bool {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	end := time.Now().Add(time.Duration(seconds) * time.Second)
	for {
		select {
		case <-ctx.Done():
			fmt.Print("\r")
			return true
		case now := <-ticker.C:
			rem := int(end.Sub(now).Seconds())
			if rem < 0 {
				rem = 0
			}
			fmt.Printf("\rNext run in %4ds  ", rem)
			if rem == 0 {
				fmt.Print("\r")
				return false
			}
		}
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	interval := getenvInt("INTERVAL", 86400)
	maxMul := getenvInt("MAX_MULTIPLIER", 10)
	showCountdown := getenvBool("COUNTDOWN", false)

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: INTERVAL=... MAX_MULTIPLIER=... COUNTDOWN=true ./scheduler /path/to/cmd [args...]")
		os.Exit(2)
	}

	seedMathRand()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		// create cancellable context for this run
		ctx, cancel := context.WithCancel(context.Background())

		// per-run signal watcher: only goroutine reading sigCh while this run is active
		go func() {
			select {
			case sig := <-sigCh:
				// forward first signal to the process group
				_ = syscall.Kill(0, sig.(syscall.Signal))
				// cancel the run so runOnce sees ctx.Done()
				cancel()
				// escalate on subsequent signals
				go func() {
					for range sigCh {
						_ = syscall.Kill(0, syscall.SIGKILL)
					}
				}()
			case <-ctx.Done():
				// run finished normally; stop watching signals
			}
		}()

		// run the command; returns when child exits (normally or due to cancellation)
		exitCode := runOnce(ctx, args)

		// ensure context is cancelled and watcher exits
		cancel()

		// if program exited uncleanly, exit loop
		// XXX: halts on program crash. should this save an error log and restart?
		if exitCode != 0 {
			fmt.Fprintf(os.Stderr, "exit: %d", exitCode)
			break
		}

		// compute next delay...
		next, err := expDelaySeconds(float64(interval), maxMul)
		if err != nil {
			fmt.Fprintf(os.Stderr, "delay draw error: %v\n", err)
			break
		}

		// watch for signals
		cdCtx, cdCancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-sigCh:
				// cancel timers
				cdCancel()
			case <-cdCtx.Done():
			}
		}()

		// display countdown until next invocation
		if showCountdown {
			cancelled := countdownLoop(cdCtx, next)
			cdCancel()
			if cancelled {
				break
			}
		} else {
			// use a timer quietly
			timer := time.NewTimer(time.Duration(next) * time.Second)
			fmt.Printf("Next run in %ds\n", next)
			select {
			case <-timer.C:
				cdCancel()
			case <-cdCtx.Done():
				timer.Stop()
				return
			}
		}
	}
}
