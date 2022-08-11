package main

//
// Code to generate a new experiment flow.
//

import (
	_ "embed"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/AlecAivazis/survey/v2"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

// Implements interactively generating a new experiment.
type NewTaskCommand struct{}

// Information about a task to autogenerate.
type TaskInfo struct {
	// The task struct's name.
	StructName string

	// Description contains the description.
	Description string

	// The task template.
	Template string
}

// Called by the CLI parser
func (c *NewTaskCommand) Run(*cobra.Command, []string) {
	printf("\n")
	printf("Welcome! This command will help you to automatically generate a task\n")
	printf("to include it into an existing OONI experiment!\n")
	print("\n")

	experimentName := getExperimentPackageName()
	info := getTaskInfo()

	generateTaskGo(experimentName, info)
}

// Obtains the experiment's package name
func getExperimentPackageName() string {
	printf("Please, enter the name of the Go package under ./internal/experiment` for\n")
	printf("which you want to autogenerate a new task.\n")
	print("\n")
	prompt := &survey.Input{
		Message: "Experiment's package name:",
	}
	var experiment string
	err := survey.AskOne(prompt, &experiment)
	runtimex.PanicOnError(err, "survey.AskOne failed")
	return experiment
}

// Obtains information about the task to generate.
func getTaskInfo() *TaskInfo {
	return &TaskInfo{
		StructName:  getTaskStructName(),
		Description: getTaskDescription(),
		Template:    getTaskTemplate(),
	}
}

// Returns the name of the task struct.
func getTaskStructName() string {
	prompt := &survey.Input{
		Message: "Task struct name (e.g., 'Datacenter'):",
	}
	var name string
	err := survey.AskOne(prompt, &name)
	runtimex.PanicOnError(err, "survey.AskOne failed")
	return name
}

// Returns the task's description
func getTaskDescription() string {
	prompt := &survey.Input{
		Message: "Short documentation for this task:",
	}
	var docs string
	err := survey.AskOne(prompt, &docs)
	runtimex.PanicOnError(err, "survey.AskOne failed")
	return docs
}

//go:embed "task/httpget.go.txt"
var httpGetTemplate string

//go:embed "task/systemresolver.go.txt"
var systemResolverTemplate string

//go:embed "task/tcpconnect.go.txt"
var tcpConnectTemplate string

//go:embed "task/tlshandshake.go.txt"
var tlsHandshakeTemplate string

// The list of known tasks
var knownTasks = map[string]string{
	"http-get":        httpGetTemplate,
	"system-resolver": systemResolverTemplate,
	"tcp-connect":     tcpConnectTemplate,
	"tls-handshake":   tlsHandshakeTemplate,
}

// Names of known tasks
var knownTaskNames []string

// Autogenerates the names of the tasks.
func init() {
	for name := range knownTasks {
		knownTaskNames = append(knownTaskNames, name)
	}
}

// Returns the task template to use.
func getTaskTemplate() string {
	var name string
	prompt := &survey.Select{
		Message: "Choose a task you would like to generate:",
		Options: knownTaskNames,
	}
	err := survey.AskOne(prompt, &name)
	runtimex.PanicOnError(err, "survey.AskOne failed")
	return name
}

// Generates code for the new task.
func generateTaskGo(experiment string, info *TaskInfo) {
	name := "task" + strings.ToLower(info.StructName) + ".go"
	fullpath := filepath.Join("internal", "experiment", experiment, name)
	tmpl := template.Must(template.New("T1").Parse(knownTasks[info.Template]))
	mapping := map[string]string{
		"Package":     experiment,
		"StructName":  info.StructName,
		"Template":    info.Template,
		"Description": info.Description,
	}
	writeTemplate(fullpath, tmpl, mapping)
	gofmt(fullpath)
}
