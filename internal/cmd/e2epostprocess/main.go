package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/execabs"
)

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Measurement contains a OONI measurement.
type Measurement struct {
	ReportID        string  `json:"report_id"`
	Input           *string `json:"input"`
	SoftwareName    string  `json:"software_name"`
	SoftwareVersion string  `json:"software_version"`
}

func main() {
	backend := flag.String("backend", "https://api.ooni.io/", "Backend to use")
	expected := flag.Int("expected", 0, "Expected number of measurement files")
	flag.Parse()
	if *expected <= 0 {
		log.Fatal("You MUST specify `-expected N`")
	}
	// based off https://flaviocopes.com/go-list-files/
	var files []string
	outputdir := "./E2E/"
	err := filepath.Walk(outputdir, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	fatalOnError(err)
	if len(files) <= 0 {
		log.Fatal("no files to process?!")
	}
	var found int
	for _, file := range files {
		data, err := os.ReadFile(file)
		fatalOnError(err)
		measurements := bytes.Split(data, []byte("\n"))
		for _, measurement := range measurements {
			if len(measurement) <= 0 {
				continue
			}
			var entry Measurement
			err = json.Unmarshal(measurement, &entry)
			fatalOnError(err)
			log.Printf("processing: %+v", entry)
			options := []string{
				"run",
				"./internal/cmd/apitool",
				"-backend", *backend,
				"-mode", "meta",
				"-report-id", entry.ReportID,
			}
			found++
			if entry.Input != nil {
				options = append(options, "-input")
				options = append(options, *entry.Input)
			}
			log.Printf("run: go %s", strings.Join(options, " "))
			cmd := execabs.Command("go", options...)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			err = cmd.Run()
			fatalOnError(err)
		}
	}
	if found != *expected {
		log.Fatalf("expected %d measurements, found %d measurements", *expected, found)
	}
}
