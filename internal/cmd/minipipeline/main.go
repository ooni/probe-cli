// Command minipipeline loads in input Web Connectivity measurements
// and applies the probe's detection heuristics on them again.
//
// By doing that, we can iterate more quickly on improving heuristics.
package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func main() {
	for _, arg := range os.Args[1:] {
		processfile(arg)
	}
}

func processfile(arg string) {
	filep, err := os.Open(arg)
	if err != nil {
		log.WithError(err).Fatal("os.Open failed")
	}
	defer filep.Close()
	scanner := bufio.NewScanner(filep)
	buf := make([]byte, 1<<23)
	scanner.Buffer(buf, len(buf))
	index := 1
	for scanner.Scan() {
		processmeasurement(scanner.Bytes(), index)
		index++
	}
	if err := scanner.Err(); err != nil {
		log.WithError(err).Fatal("scanner.Err failed")
	}
}

func processmeasurement(mraw []byte, index int) {
	var m model.Measurement
	if err := json.Unmarshal(mraw, &m); err != nil {
		log.WithError(err).Fatal("json.Unmarshal failed")
	}
	if m.TestName != "web_connectivity" {
		return
	}
	tkraw, err := json.Marshal(m.TestKeys)
	if err != nil {
		log.WithError(err).Fatal("json.Marshal failed")
	}
	processtestkeys(tkraw, string(m.Input), index)
}

func processtestkeys(tkraw []byte, input string, index int) {
	var tk webconnectivity.TestKeys
	if err := json.Unmarshal(tkraw, &tk); err != nil {
		log.WithError(err).Fatal("json.Unmarshal failed")
	}
	newtk := &webconnectivity.TestKeys{
		NetworkEvents:        tk.NetworkEvents,
		DNSWoami:             tk.DNSWoami,
		DoH:                  tk.DoH,
		Do53:                 tk.Do53,
		Queries:              tk.Queries,
		Requests:             tk.Requests,
		TCPConnect:           tk.TCPConnect,
		TLSHandshakes:        tk.TLSHandshakes,
		ControlRequest:       tk.ControlRequest,
		Control:              tk.Control,
		ControlFailure:       tk.ControlFailure,
		XDNSFlags:            0,
		DNSExperimentFailure: nil,
		DNSConsistency:       "",
		XBlockingFlags:       0,
		BodyLengthMatch:      nil,
		HeadersMatch:         nil,
		StatusCodeMatch:      nil,
		TitleMatch:           nil,
		Blocking:             nil,
		Accessible:           nil,
	}
	reprocesstk(newtk, input, index)
}

func reprocesstk(tk *webconnectivity.TestKeys, input string, index int) {
	log.Infof("\n\n\n")
	log.Infof("Input: %s", input)
	log.Infof("Idx: %d", index)
	tk.Finalize(log.Log)
}
