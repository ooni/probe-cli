package ptx

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ErrWrongBridgeType indicates that the parser we're currently
// using does not recognize the specified bridge type.
var ErrWrongBridgeType = errors.New("tunnel: wrong bridge type")

// ErrParseBridgeLine is an error when parsing the bridge line.
var ErrParseBridgeLine = errors.New("tunnel: cannot parse bridge line")

// OBFS4BridgeLineParser parses a bridge line to an OBFS4 data
// structure, or returns an error. We return the ErrWrongBridgeType
// in case the bridge type does not match the expected bridge type. We
// also return ErrParseBridgeLine on parse error. The expected format
// for an obfs4 bridge line is the one returned by the
// https://bridges.torproject.org website. The following
// is the pattern recognized by this function:
//
//     obfs4 <address>:<port> <fingerprint> cert=<cert> iat-mode=<mode>
//
// Note that the relative order of `cert` and `iat-mode` does
// not matter, but we expect both options to be present.
//
// We also recognize the case where the line starts with the
// string "Bridge", to support the way in which bridges are
// specified in the `tor` configuration file.
type OBFS4BridgeLineParser struct {
	// BridgeLine contains the bridge line to parse.
	BridgeLine string

	// DataDir contains the data directory.
	DataDir string
}

// Parse parses the OBFS4BridgeLine into an OBFS4 structure or an error.
func (p *OBFS4BridgeLineParser) Parse() (*OBFS4Dialer, error) {
	vals := strings.Split(p.BridgeLine, " ")
	blp := &obfs4BridgeLineParserCtx{
		bridgeKeyword: make(chan *obfs4BridgeLineParserState),
		bridgeType:    make(chan *obfs4BridgeLineParserState),
		endpoint:      make(chan *obfs4BridgeLineParserState),
		fingerprint:   make(chan *obfs4BridgeLineParserState),
		options:       make(chan *obfs4BridgeLineParserState),
		nextOptions:   make(chan *obfs4BridgeLineParserState),
		err:           make(chan error),
		result:        make(chan *OBFS4Dialer),
		wg:            &sync.WaitGroup{},
	}
	launch := func(f func()) {
		blp.wg.Add(1) // count goro as running
		go f()
	}
	launch(blp.parseBridgeKeyword)
	launch(blp.parseBridgeType)
	launch(blp.parseEndpoint)
	launch(blp.parseFingerprint)
	launch(blp.parseOptions)
	launch(blp.parseNextOptions)
	blp.bridgeKeyword <- &obfs4BridgeLineParserState{ // kick off
		vals: vals,
		o4:   &OBFS4Dialer{DataDir: p.DataDir},
	}
	var (
		err    error
		result *OBFS4Dialer
	)
	select {
	case err = <-blp.err:
	case result = <-blp.result:
		return result, nil
	}
	close(blp.bridgeKeyword)
	close(blp.bridgeType)
	close(blp.endpoint)
	close(blp.fingerprint)
	close(blp.options)
	close(blp.nextOptions)
	blp.wg.Wait() // join the goros
	return result, err
}

// obfs4BridgeLineParserState contains the parser state (i.e., the
// "piece" that is to be worked on by the "stations").
type obfs4BridgeLineParserState struct {
	// vals contains the not-parsed-yet tokens
	vals []string

	// o4 contains the output structure
	o4 *OBFS4Dialer
}

// obfs4BridgeLineParserCtx contains the parser context (i.e., the
// context grouping the variables used by the parse "stations").
type obfs4BridgeLineParserCtx struct {
	// bridgeKeyword is the input of the parseBridgeKeyword parser.
	bridgeKeyword chan *obfs4BridgeLineParserState

	// bridgeType is the input of the parseBridgeType parser.
	bridgeType chan *obfs4BridgeLineParserState

	// endpoint is the input of the parseEndpoint parser.
	endpoint chan *obfs4BridgeLineParserState

	// fingerprint is the input for the parseFingerprint state.
	fingerprint chan *obfs4BridgeLineParserState

	// options is the input of the parseOptions parser.
	options chan *obfs4BridgeLineParserState

	// nextOptions is the input of the parseNextOptions parser.
	nextOptions chan *obfs4BridgeLineParserState

	// err is an output indicating that parsing failed.
	err chan error

	// result is an output indicating that parsing succeded.
	result chan *OBFS4Dialer

	// wg counts the number of running goroutines
	wg *sync.WaitGroup
}

// parseBridgeKeyword parses the optional "bridge" keyword.
func (p *obfs4BridgeLineParserCtx) parseBridgeKeyword() {
	defer p.wg.Done()
	for s := range p.bridgeKeyword {
		if len(s.vals) >= 1 && strings.ToLower(s.vals[0]) == "bridge" {
			s.vals = s.vals[1:] // just skip the keyword
		}
		p.bridgeType <- s
	}
}

// parseBridgeType parses the mandatory bridge type ("obfs4").
func (p *obfs4BridgeLineParserCtx) parseBridgeType() {
	defer p.wg.Done()
	for s := range p.bridgeType {
		if len(s.vals) < 1 {
			p.err <- fmt.Errorf("%w: missing bridge type", ErrParseBridgeLine)
			continue
		}
		if s.vals[0] != "obfs4" {
			p.err <- fmt.Errorf(
				"%w: expected 'obfs4', found '%s'", ErrWrongBridgeType, s.vals[0])
			continue
		}
		s.vals = s.vals[1:]
		p.endpoint <- s
	}
}

// parseEndpoint parses the mandatory bridge endpoint. We expect the
// endpoint to be like 1.2.3.4:5678 or like [::1:ef:3:4]:5678.
func (p *obfs4BridgeLineParserCtx) parseEndpoint() {
	defer p.wg.Done()
	for s := range p.endpoint {
		if len(s.vals) < 1 {
			p.err <- fmt.Errorf("%w: missing bridge endpoint", ErrParseBridgeLine)
			continue
		}
		if _, _, err := net.SplitHostPort(s.vals[0]); err != nil {
			p.err <- fmt.Errorf("%w: %s", ErrParseBridgeLine, err.Error())
			continue
		}
		s.o4.Address = s.vals[0]
		s.vals = s.vals[1:]
		p.fingerprint <- s
	}
}

// parseFingerprint parses the fingerprint.
func (p *obfs4BridgeLineParserCtx) parseFingerprint() {
	defer p.wg.Done()
	for s := range p.fingerprint {
		if len(s.vals) < 1 {
			p.err <- fmt.Errorf("%w: missing bridge fingerprint", ErrParseBridgeLine)
			continue
		}
		re := regexp.MustCompile("^[A-Fa-f0-9]{40}$")
		if !re.MatchString(s.vals[0]) {
			p.err <- fmt.Errorf("%w: invalid bridge fingerprint", ErrParseBridgeLine)
			continue
		}
		s.o4.Fingerprint = s.vals[0]
		s.vals = s.vals[1:]
		p.options <- s
	}
}

// parseOptions parses the options.
func (p *obfs4BridgeLineParserCtx) parseOptions() {
	defer p.wg.Done()
	for s := range p.options {
		if len(s.vals) < 1 {
			if s.o4.Cert == "" {
				p.err <- fmt.Errorf("%w: missing bridge cert", ErrParseBridgeLine)
				continue
			}
			if s.o4.IATMode == "" {
				p.err <- fmt.Errorf("%w: missing bridge iat-mode", ErrParseBridgeLine)
				continue
			}
			p.result <- s.o4
			continue
		}
		v := s.vals[0]
		s.vals = s.vals[1:]
		if strings.HasPrefix(v, "cert=") {
			v = v[len("cert="):]
			cert := v + "=="
			if _, err := base64.StdEncoding.DecodeString(cert); err != nil {
				p.err <- fmt.Errorf(
					"%w: cannot parse cert: %s", ErrParseBridgeLine, err.Error())
				continue
			}
			s.o4.Cert = v
			p.nextOptions <- s // avoid self deadlock
			continue
		}
		if strings.HasPrefix(v, "iat-mode") {
			v = v[len("iat-mode="):]
			if _, err := strconv.Atoi(v); err != nil {
				p.err <- fmt.Errorf(
					"%w: cannot parse iat-mode: %s", ErrParseBridgeLine, err.Error())
				continue
			}
			s.o4.IATMode = v
			p.nextOptions <- s // avoid self deadlock
			continue
		}
		p.err <- fmt.Errorf("%w: invalid option: %s", ErrParseBridgeLine, v)
	}
}

// parseOptions parses the options.
func (p *obfs4BridgeLineParserCtx) parseNextOptions() {
	defer p.wg.Done()
	for s := range p.nextOptions {
		p.options <- s
	}
}
