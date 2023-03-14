// Package testlists contains code to walk through the test lists.
package testlists

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/schollz/progressbar/v3"
)

// Entry is an entry of a test list.
type Entry struct {
	// File is the file from which we are reading.
	File string

	// Line is the line within the input file.
	Line int64

	// URL is the URL.
	URL string

	// CategoryCode is the category code.
	CategoryCode string

	// CategoryDescription describes the category.
	CategoryDescription string

	// DateAdded is when the entry was added.
	DateAdded string

	// Source is who added the entry.
	Source string

	// Notes contains free-form textual notes.
	Notes string
}

// Generator is a function that posts each entry within each file in the
// citizenlab/test-lists on the given channel. This function will close the
// och channel when it has finished reading all the test-lists. This
// function calls [runtimex.PanicOnError] in case an error occurs.
func Generator(wg *sync.WaitGroup, testListsDir string, och chan<- *Entry) {
	// logging
	log.Debugf("generator for %s... running", testListsDir)
	defer log.Debugf("generator for %s... done", testListsDir)

	// synchronize with the parent
	defer wg.Done()

	// notify the reader that we're done
	defer close(och)

	// create regexp for filtering the entries we care about
	validator := regexp.MustCompile(`^([a-z]{2}|cis|global)\.csv$`)

	// read the directory containing the lists
	entries := runtimex.Try1(os.ReadDir(testListsDir))
	for _, entry := range entries {
		// make sure we skip everything that isn't a regular file
		if !entry.Type().IsRegular() {
			continue
		}

		// make sure we only include the lists that matter
		if !validator.MatchString(entry.Name()) {
			continue
		}

		// collect all the entries
		all := collect(filepath.Join(testListsDir, entry.Name()))

		// emit all the entries
		emit(entry.Name(), all, och)
	}
}

// collect collects all the test list entries.
func collect(filepath string) (all []*Entry) {
	// open file and create CSV reader
	filep := runtimex.Try1(os.Open(filepath))
	reader := csv.NewReader(filep)

	// loop through all entries
	var lineno int64
	for {
		// read the current entry
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		runtimex.PanicOnError(err, "reader.Read")
		// this record seems malformed but in theory this
		// cannot happen because the csv library should return
		// an error in case we see a short record.
		runtimex.Assert(len(record) == 6, "unexpected record length")

		// skip the first line, which contains the headers
		lineno++
		if lineno == 1 {
			continue
		}

		// assemble and add the corresponding test list entry
		entry := &Entry{
			File:                filepath,
			Line:                lineno,
			URL:                 record[0],
			CategoryCode:        record[1],
			CategoryDescription: record[2],
			DateAdded:           record[3],
			Source:              record[4],
			Notes:               record[5],
		}
		all = append(all, entry)
	}

	// return all the collected entries
	return
}

// emit emits all the entries while incrementing a progessbar
func emit(filepath string, all []*Entry, och chan<- *Entry) {
	bar := progressbar.NewOptions64(
		int64(len(all)),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription(filepath),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSetWriter(os.Stdout),
	)
	for _, entry := range all {
		bar.Add(1)
		och <- entry
	}
}
