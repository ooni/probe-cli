package main

import (
	"io/ioutil"
	"log"
)

func main() {
	// read the content of GOVERSION
	data, err := ioutil.ReadFile("GOVERSION")
	if err != nil {
		log.Fatal(err)
	}

	// strip trailing newlines
	for len(data) > 0 && data[len(data)-1] == '\r' || data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	// run the specified version of go
	Run("go" + string(data))
}
