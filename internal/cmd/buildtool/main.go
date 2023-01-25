package main

//
// Main
//

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "buildtool",
		Short: "Tool for building ooniprobe, miniooni, etc.",
	}
	root.AddCommand(darwinSubcommand())
	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = true
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logHandler}

	go func() {
		if r := recover(); r != nil {
			log.Fatalf("%+v", r)
		}
	}()
	err := root.Execute()
	runtimex.PanicOnError(err, "root.Execute")
}
