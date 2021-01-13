package darwin

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/ooni/probe-cli/internal/utils"
	"github.com/ooni/probe-engine/cmd/jafar/shellx"
)

// Manager allows to start/stop running periodically.
type Manager struct{}

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

func (Manager) mustNotHavePlist() error {
	if utils.FileExists(plistPath) {
		// This is not atomic. Do we need atomicity here?
		return errors.New("periodic: service already registered")
	}
	return nil
}

func (Manager) writePlist() error {
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
	return ioutil.WriteFile(plistPath, out.Bytes(), 0644)
}

func (Manager) start() error {
	if err := shellx.Run("launchctl", "enable", serviceTarget); err != nil {
		return err
	}
	return shellx.Run("launchctl", "bootstrap", domainTarget, plistPath)
}

// Start starts running periodically.
func (m Manager) Start() error {
	operations := []func() error{m.mustNotHavePlist, m.writePlist, m.start}
	for _, op := range operations {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

func (Manager) stop() error {
	return shellx.Run("launchctl", "bootout", serviceTarget)
}

func (Manager) removeFile() error {
	// TODO(bassosimone): maybe we should ignore ENOENT.
	return os.Remove(plistPath)
}

// Stop stops running periodically.
func (m Manager) Stop() error {
	operations := []func() error{m.stop, m.removeFile}
	for _, op := range operations {
		if err := op(); err != nil {
			// nothing: we want this command to be idempotent
			// and maybe we can achieve this by filtering more
			// carefully the errors that are returned?
			// TODO(bassosimone): improve
		}
	}
	return nil
}
