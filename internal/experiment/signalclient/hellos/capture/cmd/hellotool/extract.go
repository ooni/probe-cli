package main

//
// extract, normalize and same: turning captures into the ClientHello files hellos.go embeds.
//

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

// tcpStreams is each TCP connection's client stream, then its server stream, in the order the
// connections were first seen.
func tcpStreams(packets []packet) [][]byte {
	var out [][]byte
	for _, conn := range tcpConnections(packets) {
		up, _ := conn.up.bytes()
		down, _ := conn.down.bytes()
		out = append(out, up, down)
	}
	return out
}

// findHello returns the first ClientHello sent to sni in a capture, normalized, and its fields.
func findHello(capturePath, sni string) ([]byte, *helloFields, error) {
	c, err := readCapture(capturePath)
	if err != nil {
		return nil, nil, err
	}
	for _, stream := range tcpStreams(c.packets) {
		hs := clientHello(stream)
		if hs == nil {
			continue
		}
		fields, err := parseHello(hs)
		if err != nil || fields.sni != sni {
			continue
		}
		hs, err = normalize(hs)
		return hs, fields, err
	}
	return nil, nil, fmt.Errorf("no ClientHello for %s in %s", sni, capturePath)
}

// extract writes the first ClientHello sent to sni in a capture, normalized, to out, and prints
// its size, ALPN, JA4 and JA3.
func extract(capturePath, sni, out string) error {
	hs, fields, err := findHello(capturePath, sni)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, hs, 0o644); err != nil {
		return err
	}
	alpn := printable(strings.Join(fields.alpn, ","))
	if alpn == "" {
		alpn = "(none)"
	}
	fmt.Printf("%s: %d bytes, SNI %s, ALPN %s, JA4 %s, JA3 %s\n", out, len(hs), sni, alpn, fields.ja4(), fields.ja3())
	return nil
}

// normalizeFiles zeroes the random values of existing ClientHello files.
func normalizeFiles(paths []string) error {
	for _, path := range paths {
		hs, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if hs, err = normalize(hs); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := os.WriteFile(path, hs, 0o644); err != nil {
			return err
		}
		fmt.Printf("%s: normalized\n", path)
	}
	return nil
}

// helloShape is a ClientHello file without what depends on the server's name: the SNI, and the
// padding extension, whose length follows the SNI's.
type helloShape struct {
	beforeExtensions string            // version, random, session id, ciphers, compression
	extensions       []uint16          // in order
	extensionData    map[uint16]string // except SNI and padding
	paddedLength     int               // the whole hello's length when it is padded, else 0
}

func shapeOf(hs []byte) (*helloShape, error) {
	f, err := parseHello(hs)
	if err != nil {
		return nil, err
	}
	s := &helloShape{beforeExtensions: string(hs[4:f.extensionsStart]), extensions: f.extensions,
		extensionData: map[uint16]string{}}
	for typ, data := range f.extensionData {
		if typ != extServerName && typ != extPadding {
			s.extensionData[typ] = string(data)
		}
	}
	if _, padded := f.extensionData[extPadding]; padded {
		s.paddedLength = len(hs)
	}
	return s, nil
}

func (s *helloShape) equal(o *helloShape) bool {
	if s.beforeExtensions != o.beforeExtensions || s.paddedLength != o.paddedLength ||
		!slices.Equal(s.extensions, o.extensions) || len(s.extensionData) != len(o.extensionData) {
		return false
	}
	for typ, data := range s.extensionData {
		if o.extensionData[typ] != data {
			return false
		}
	}
	return true
}

// sameHello checks that ClientHello files are one hello: identical but for the SNI and the
// padding's length. It reports each file that differs from the first.
func sameHello(paths []string) error {
	if len(paths) == 0 {
		return errors.New("no files")
	}
	var first *helloShape
	differ := 0
	for i, path := range paths {
		hs, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s, err := shapeOf(hs)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if i == 0 {
			first = s
		} else if !s.equal(first) {
			fmt.Fprintf(os.Stderr, "hellotool same: %s differs from %s (beyond the SNI and padding)\n", path, paths[0])
			differ++
		}
	}
	if differ > 0 {
		return fmt.Errorf("%d of %d files differ", differ, len(paths))
	}
	return nil
}
