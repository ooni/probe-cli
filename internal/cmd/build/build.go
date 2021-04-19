// Command build builds ooniprobe.
package main

import (
	"github.com/alecthomas/kong"
	"github.com/apex/log"
)

// GlobalFlags contains global flags.
type GlobalFlags struct {
	// Verbose runs a verbose build.
	Verbose bool `short:"v" help:"Verbose mode."`
}

// CLI contains CLI flags.
type CLI struct {
	// GlobalFlags contains the global flags.
	GlobalFlags

	// Android imlements the android command.
	Android AndroidCmd `cmd:"android" help:"Build Android library (aka oonimkall)."`
}

// must fails if there is an error.
func must(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

// mustString is like must but with an extra string argument.
func mustString(str string, err error) string {
	must(err)
	return str
}

func main() {
	cli := CLI{}
	parsed := kong.Parse(&cli,
		kong.Name("go run ./internal/cmd/build"),
		kong.Description("OONI Probe build script"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	err := parsed.Run(&cli.GlobalFlags)
	parsed.FatalIfErrorf(err)
}
