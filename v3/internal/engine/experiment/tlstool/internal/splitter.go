package internal

import (
	"bytes"
	"math/rand"
	"time"
)

// SNISplitter splits input such that SNI is splitted across
// a bunch of different output buffers.
func SNISplitter(input []byte, sni []byte) (output [][]byte) {
	idx := bytes.Index(input, sni)
	if idx < 0 {
		output = append(output, input)
		return
	}
	output = append(output, input[:idx])
	// TODO(bassosimone): splitting every three bytes causes
	// a bunch of Unicode chatacters (e.g., in Chinese) to be
	// sent as part of the same segment. Is that OK?
	const segmentsize = 3
	var buf []byte
	for _, chr := range input[idx : idx+len(sni)] {
		buf = append(buf, chr)
		if len(buf) == segmentsize {
			output = append(output, buf)
			buf = nil
		}
	}
	if len(buf) > 0 {
		output = append(output, buf)
		buf = nil
	}
	output = append(output, input[idx+len(sni):])
	return
}

// Splitter84rest segments the specified buffer into three
// sub-buffers containing respectively 8 bytes, 4 bytes, and
// the rest of the buffer. This segment technique has been
// described by Kevin Bock during the Internet Measurements
// Village 2020: https://youtu.be/ksojSRFLbBM?t=1140.
func Splitter84rest(input []byte) (output [][]byte) {
	if len(input) <= 12 {
		output = append(output, input)
		return
	}
	output = append(output, input[:8])
	output = append(output, input[8:12])
	output = append(output, input[12:])
	return
}

// Splitter3264rand splits the specified buffer at a random
// offset between 32 and 64 bytes. This is the methodology used
// by github.com/Jigsaw-Code/outline-go-tun2socks.
func Splitter3264rand(input []byte) (output [][]byte) {
	if len(input) <= 64 {
		output = append(output, input)
		return
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	offset := rnd.Intn(32) + 32
	output = append(output, input[:offset])
	output = append(output, input[offset:])
	return
}
