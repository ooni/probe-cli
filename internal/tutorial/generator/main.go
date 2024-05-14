// Command generator generates or re-generates the tutorial chapters. You
// should run this command like `go run ./generator`.
package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

// writeString writes a string on the given writer. If there
// is a write error, this function will call log.Fatal.
func writeString(w io.Writer, s string) {
	if _, err := io.WriteString(w, s); err != nil {
		log.Fatal(err)
	}
}

// gen1 generates a single file within a chapter.
func gen1(destfile io.Writer, filepath string) {
	srcfile, err := os.Open(filepath) // #nosec G304 - this is working as intended
	if err != nil {
		log.Fatal(err)
	}
	defer srcfile.Close()
	scanner := bufio.NewScanner(srcfile)
	var started bool
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.Trim(line, " \t\r\n")
		if trimmed == "// -=-=- StopHere -=-=-" {
			started = false
			continue
		}
		if trimmed == "// -=-=- StartHere -=-=-" {
			started = true
			continue
		}
		if !started {
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			if strings.HasPrefix(trimmed, "// ") {
				trimmed = trimmed[3:]
			} else {
				trimmed = trimmed[2:]
			}
			writeString(destfile, trimmed+"\n")
			continue
		}
		writeString(destfile, line+"\n")
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// gen generates or re-generates a chapter. The dirpath argument
// is the path to the directory that contains a chapter. The files
// arguments contains the source file names to process. We will process
// files using the specified order. Note that files names are not
// paths, just file names, e.g.,
//
//	gen("./experiment/torsf/chapter01", "main.go")
func gen(dirpath string, files ...string) {
	readme := path.Join(dirpath, "README.md")
	destfile, err := os.Create(path.Join(readme)) // #nosec G304 - this is working as intended
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := destfile.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	for _, file := range files {
		gen1(destfile, path.Join(dirpath, file))
	}
}

// gentorsf generates the torsf chapters.
func gentorsf() {
	prefix := path.Join(".", "experiment", "torsf")
	gen(path.Join(prefix, "chapter01"), "main.go")
	gen(path.Join(prefix, "chapter02"), "main.go", "torsf.go")
	gen(path.Join(prefix, "chapter03"), "torsf.go")
	gen(path.Join(prefix, "chapter04"), "torsf.go")
}

// genmeasurex generates the measurex chapters.
func genmeasurex() {
	prefix := path.Join(".", "measurex")
	gen(path.Join(prefix, "chapter01"), "main.go")
	gen(path.Join(prefix, "chapter02"), "main.go")
	gen(path.Join(prefix, "chapter03"), "main.go")
	gen(path.Join(prefix, "chapter04"), "main.go")
	gen(path.Join(prefix, "chapter05"), "main.go")
	gen(path.Join(prefix, "chapter06"), "main.go")
	gen(path.Join(prefix, "chapter07"), "main.go")
	gen(path.Join(prefix, "chapter08"), "main.go")
	gen(path.Join(prefix, "chapter09"), "main.go")
	gen(path.Join(prefix, "chapter10"), "main.go")
	gen(path.Join(prefix, "chapter11"), "main.go")
	gen(path.Join(prefix, "chapter12"), "main.go")
	gen(path.Join(prefix, "chapter13"), "main.go")
	gen(path.Join(prefix, "chapter14"), "main.go")
}

// gennetxlite generates the netxlite chapters.
func gennetxlite() {
	prefix := path.Join(".", "netxlite")
	gen(path.Join(prefix, "chapter01"), "main.go")
	gen(path.Join(prefix, "chapter02"), "main.go")
	gen(path.Join(prefix, "chapter03"), "main.go")
	gen(path.Join(prefix, "chapter04"), "main.go")
	gen(path.Join(prefix, "chapter05"), "main.go")
	gen(path.Join(prefix, "chapter06"), "main.go")
	gen(path.Join(prefix, "chapter07"), "main.go")
	gen(path.Join(prefix, "chapter08"), "main.go")
}

// gendslx generates the dslx chapters.
func gendslx() {
	prefix := path.Join(".", "dslx")
	gen(path.Join(prefix, "chapter02"), "main.go")
}

func main() {
	gentorsf()
	genmeasurex()
	gennetxlite()
	gendslx()
}
