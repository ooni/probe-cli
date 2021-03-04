package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/execabs"
)

func writefile(name string, sb *strings.Builder) {
	filep, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := fmt.Fprint(filep, sb.String()); err != nil {
		log.Fatal(err)
	}
	if err := filep.Close(); err != nil {
		log.Fatal(err)
	}
	cmd := execabs.Command("go", "fmt", name)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
