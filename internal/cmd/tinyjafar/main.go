// Command tinyjafar implements a subset of the CLI flags of the original jafar tool. Because several
// tutorials mention some jafar commands, we want to have a tiny tool to support exploration.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/apex/log"
	"github.com/google/shlex"
	"github.com/ooni/probe-cli/v3/internal/flagx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// config contains tinyjafar's configuration.
type config struct {
	dropIP          flagx.StringArray
	dropKeywordHex  flagx.StringArray
	dropKeyword     flagx.StringArray
	dryRun          bool
	resetIP         flagx.StringArray
	resetKeywordHex flagx.StringArray
	resetKeyword    flagx.StringArray
}

func (cfg *config) initFlags(fset *flag.FlagSet) {
	fset.Var(&cfg.dropIP, "iptables-drop-ip", "Drop traffic to the specified IP address")
	fset.Var(&cfg.dropKeywordHex, "iptables-drop-keyword-hex", "Drop traffic containing the specified keyword in hex")
	fset.Var(&cfg.dropKeyword, "iptables-drop-keyword", "Drop traffic containing the specified keyword")
	fset.BoolVar(&cfg.dryRun, "dry-run", false, "print which commands we would execute")
	fset.Var(&cfg.resetIP, "iptables-reset-ip", "Reset TCP/IP traffic to the specified IP address")
	fset.Var(&cfg.resetKeywordHex, "iptables-reset-keyword-hex", "Reset TCP/IP traffic containing the specified keyword in hex")
	fset.Var(&cfg.resetKeyword, "iptables-reset-keyword", "Reset TCP/IP traffic containing the specified keyword")
}

// cmd is a cmd to execute
type cmd struct {
	argv []string
}

// cmdSet contains the commands to execute. The zero value is invalid
// and you must construct using the [newCmdSet] factory.
type cmdSet struct {
	setup   []*cmd
	cleanup []*cmd
}

func newCmdSet() *cmdSet {
	c := &cmdSet{}

	c.addSetupCmd("iptables -N JAFAR_INPUT")
	c.addSetupCmd("iptables -N JAFAR_OUTPUT")
	c.addSetupCmd("iptables -t nat -N JAFAR_NAT_OUTPUT")
	c.addSetupCmd("iptables -I OUTPUT -j JAFAR_OUTPUT")
	c.addSetupCmd("iptables -I INPUT -j JAFAR_INPUT")
	c.addSetupCmd("iptables -t nat -I OUTPUT -j JAFAR_NAT_OUTPUT")

	addCleanupCmd := func(argv string) {
		c.cleanup = append(c.cleanup, &cmd{runtimex.Try1(shlex.Split(argv))})
	}

	addCleanupCmd("iptables -D OUTPUT -j JAFAR_OUTPUT")
	addCleanupCmd("iptables -D INPUT -j JAFAR_INPUT")
	addCleanupCmd("iptables -t nat -D OUTPUT -j JAFAR_NAT_OUTPUT")
	addCleanupCmd("iptables -F JAFAR_INPUT")
	addCleanupCmd("iptables -X JAFAR_INPUT")
	addCleanupCmd("iptables -F JAFAR_OUTPUT")
	addCleanupCmd("iptables -X JAFAR_OUTPUT")
	addCleanupCmd("iptables -t nat -F JAFAR_NAT_OUTPUT")
	addCleanupCmd("iptables -t nat -X JAFAR_NAT_OUTPUT")

	return c
}

func (c *cmdSet) addSetupCmd(argv string) {
	c.setup = append(c.setup, &cmd{runtimex.Try1(shlex.Split(argv))})
}

func (c *cmdSet) handleDropIP(cfg *config) {
	for _, ipAddr := range cfg.dropIP {
		c.addSetupCmd(fmt.Sprintf("iptables -A JAFAR_OUTPUT -d '%s' -j DROP", ipAddr))
	}
}

func (c *cmdSet) handleDropKeywordHex(cfg *config) {
	for _, keyword := range cfg.dropKeywordHex {
		c.addSetupCmd(fmt.Sprintf(
			"iptables -A JAFAR_OUTPUT -m string --algo kmp --hex-string '%s' -j DROP", keyword))
	}
}

func (c *cmdSet) handleDropKeyword(cfg *config) {
	for _, keyword := range cfg.dropKeyword {
		c.addSetupCmd(fmt.Sprintf(
			"iptables -A JAFAR_OUTPUT -m string --algo kmp --string '%s' -j DROP",
			keyword,
		))
	}
}

func (c *cmdSet) handleResetIP(cfg *config) {
	for _, ipAddr := range cfg.resetIP {
		c.addSetupCmd(fmt.Sprintf(
			"iptables -A JAFAR_OUTPUT --proto tcp -d '%s' -j REJECT --reject-with tcp-reset",
			ipAddr,
		))
	}
}

func (c *cmdSet) handleResetKeywordHex(cfg *config) {
	for _, keyword := range cfg.resetKeywordHex {
		c.addSetupCmd(fmt.Sprintf(
			"iptables -A JAFAR_OUTPUT -m string --proto tcp --algo kmp --hex-string '%s' -j REJECT --reject-with tcp-reset",
			keyword,
		))
	}
}

func (c *cmdSet) handleResetKeyword(cfg *config) {
	for _, keyword := range cfg.resetKeyword {
		c.addSetupCmd(fmt.Sprintf(
			"iptables -A JAFAR_OUTPUT -m string --proto tcp --algo kmp --string '%s' -j REJECT --reject-with tcp-reset",
			keyword,
		))
	}
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT)
	mainWithArgs(os.Stdout, sigChan, os.Args[1:]...)
}

var (
	returnImmediately  = &atomic.Bool{}
	mainWithArgsCalled = &atomic.Int64{}
)

func mainWithArgs(writer io.Writer, sigChan <-chan os.Signal, args ...string) {
	if returnImmediately.Load() {
		mainWithArgsCalled.Add(1)
		return
	}

	cfg := &config{}
	fset := flag.NewFlagSet("tinyjafar", flag.ExitOnError)
	cfg.initFlags(fset)

	runtimex.Try0(fset.Parse(args))

	cs := newCmdSet()
	cs.handleDropIP(cfg)
	cs.handleDropKeywordHex(cfg)
	cs.handleDropKeyword(cfg)
	cs.handleResetIP(cfg)
	cs.handleResetKeywordHex(cfg)
	cs.handleResetKeyword(cfg)

	// with -dry-run, we're just going to print the commands we'd execute
	dryShellRun := func(logger model.Logger, command string, args ...string) error {
		_, err := fmt.Fprintf(writer, "+ %s\n", shellx.QuotedCommandLineUnsafe(command, args...))
		return err
	}
	var runSelector = map[bool]func(logger model.Logger, command string, args ...string) error{
		true:  dryShellRun,
		false: shellx.Run,
	}
	runx := runSelector[cfg.dryRun]

	for _, cmd := range cs.setup {
		runtimex.Try0(runx(log.Log, cmd.argv[0], cmd.argv[1:]...))
	}

	fmt.Fprintf(writer, "\nUse Ctrl-C to terminate\n\n")
	<-sigChan

	for _, cmd := range cs.cleanup {
		// ignoring the return value here is intentional to avoid interrupting the cleanup midway
		_ = runx(log.Log, cmd.argv[0], cmd.argv[1:]...)
	}
}
