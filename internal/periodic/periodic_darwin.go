// +build darwin

package periodic

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/utils"
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

func run(name string, arg ...string) error {
	log.Infof("exec: %s %s", name, strings.Join(arg, " "))
	return shellx.RunQuiet(name, arg...)
}

func (managerDarwin) LogShow() error {
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
		return errors.New("periodic: service already registered")
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
	if err := run("launchctl", "enable", serviceTarget); err != nil {
		return err
	}
	return run("launchctl", "bootstrap", domainTarget, plistPath)
}

// Start starts running periodically.
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
	err := run("launchctl", "bootout", serviceTarget)
	if errors.As(err, &failure) && failure.ExitCode() == int(unix.ESRCH) {
		err = nil
	}
	return err
}

func (managerDarwin) removeFile() error {
	log.Infof("exec: rm %s", plistPath)
	err := os.Remove(plistPath)
	if errors.Is(err, unix.ENOENT) {
		err = nil
	}
	return err
}

// Stop stops running periodically.
func (m managerDarwin) Stop() error {
	operations := []func() error{m.stop, m.removeFile}
	for _, op := range operations {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	register("darwin", managerDarwin{})
}
