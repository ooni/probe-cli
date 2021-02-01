package autorun

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/utils"
	"github.com/ooni/probe-engine/cmd/jafar/shellx"
	"golang.org/x/sys/unix"
)

type managerDarwin struct{}

var (
	plistPath     = os.ExpandEnv("$HOME/Library/LaunchAgents/org.ooni.cli.plist")
	domainTarget  = fmt.Sprintf("gui/%d", os.Getuid())
	serviceTarget = fmt.Sprintf("%s/org.ooni.cli", domainTarget)
)

var plistTemplate = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>org.ooni.cli</string>
    <key>KeepAlive</key>
    <false/>
    <key>RunAtLoad</key>
    <true/>
    <key>ProgramArguments</key>
    <array>
        <string>{{ .Executable }}</string>
        <string>--log-handler=syslog</string>
        <string>run</string>
        <string>unattended</string>
    </array>
    <key>StartInterval</key>
    <integer>3600</integer>
</dict>
</plist>
`

func runQuiteQuietly(name string, arg ...string) error {
	log.Infof("exec: %s %s", name, strings.Join(arg, " "))
	return shellx.RunQuiet(name, arg...)
}

func darwinVersionMajor() (int, error) {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return 0, err
	}
	v := bytes.Split(out, []byte("."))
	if len(v) != 3 {
		return 0, errors.New("cannot split version")
	}
	major, err := strconv.Atoi(string(v[0]))
	if err != nil {
		return 0, err
	}
	return major, nil
}

var errNotImplemented = errors.New(
	"autorun: command not implemented in this version of macOS")

func (managerDarwin) LogShow() error {
	major, _ := darwinVersionMajor()
	if major < 20 /* macOS 11.0 Big Sur */ {
		return errNotImplemented
	}
	return shellx.Run("log", "show", "--info", "--debug",
		"--process", "ooniprobe", "--style", "compact")
}

func (managerDarwin) LogStream() error {
	return shellx.Run("log", "stream", "--style", "compact", "--level",
		"debug", "--process", "ooniprobe")
}

func (managerDarwin) mustNotHavePlist() error {
	log.Infof("exec: test -f %s && already_registered()", plistPath)
	if utils.FileExists(plistPath) {
		// This is not atomic. Do we need atomicity here?
		return errors.New("autorun: service already registered")
	}
	return nil
}

func (managerDarwin) writePlist() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	var out bytes.Buffer
	t := template.Must(template.New("plist").Parse(plistTemplate))
	in := struct{ Executable string }{Executable: executable}
	if err := t.Execute(&out, in); err != nil {
		return err
	}
	log.Infof("exec: writePlist(%s)", plistPath)
	return ioutil.WriteFile(plistPath, out.Bytes(), 0644)
}

func (managerDarwin) start() error {
	if err := runQuiteQuietly("launchctl", "enable", serviceTarget); err != nil {
		return err
	}
	return runQuiteQuietly("launchctl", "bootstrap", domainTarget, plistPath)
}

func (m managerDarwin) Start() error {
	operations := []func() error{m.mustNotHavePlist, m.writePlist, m.start}
	for _, op := range operations {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

func (managerDarwin) stop() error {
	var failure *exec.ExitError
	err := runQuiteQuietly("launchctl", "bootout", serviceTarget)
	if errors.As(err, &failure) && failure.ExitCode() == int(unix.ESRCH) {
		err = nil
	}
	return err
}

func (managerDarwin) removeFile() error {
	log.Infof("exec: rm -f %s", plistPath)
	err := os.Remove(plistPath)
	if errors.Is(err, unix.ENOENT) {
		err = nil
	}
	return err
}

func (m managerDarwin) Stop() error {
	operations := []func() error{m.stop, m.removeFile}
	for _, op := range operations {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

func (m managerDarwin) Status() (string, error) {
	err := runQuiteQuietly("launchctl", "kill", "SIGINFO", serviceTarget)
	var failure *exec.ExitError
	if errors.As(err, &failure) {
		switch failure.ExitCode() {
		case int(unix.ESRCH):
			return StatusScheduled, nil
		case 113: // exit code when there's no plist
			return StatusStopped, nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("autorun: unexpected error: %w", err)
	}
	return StatusRunning, nil
}

func init() {
	register("darwin", managerDarwin{})
}
