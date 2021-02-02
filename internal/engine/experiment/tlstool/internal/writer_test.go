package internal_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlstool/internal"
)

func TestSleeperWriterWorksAsIntended(t *testing.T) {
	origconn := &internal.FakeConn{}
	const outdata = "deadbeefbadidea"
	conn := internal.SleeperWriter{
		Conn:  origconn,
		Delay: 1 * time.Second,
	}
	before := time.Now()
	count, err := conn.Write([]byte(outdata))
	elapsed := time.Since(before)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(outdata) {
		t.Fatal("unexpected count")
	}
	if len(origconn.WriteData) != 1 {
		t.Fatal("wrong length of written data queue")
	}
	if string(origconn.WriteData[0]) != outdata {
		t.Fatal("we did not write the right data")
	}
	if elapsed < 750*time.Millisecond {
		t.Fatalf("unexpected elapsed time: %+v", elapsed)
	}
}

func TestSplitterWriterNoSplitSuccess(t *testing.T) {
	innerconn := &internal.FakeConn{}
	conn := internal.SplitterWriter{Conn: innerconn}
	const data = "deadbeef"
	count, err := conn.Write([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if count != len(data) {
		t.Fatal("invalid count")
	}
	if len(innerconn.WriteData) != 1 {
		t.Fatal("invalid data queue")
	}
	if string(innerconn.WriteData[0]) != data {
		t.Fatal("invalid written data")
	}
}

func TestSplitterWriterNoSplitFailure(t *testing.T) {
	expected := errors.New("mocked error")
	innerconn := &internal.FakeConn{WriteError: expected}
	conn := internal.SplitterWriter{Conn: innerconn}
	const data = "deadbeef"
	count, err := conn.Write([]byte(data))
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if count != 0 {
		t.Fatal("invalid count")
	}
	if len(innerconn.WriteData) != 0 {
		t.Fatal("invalid data queue")
	}
}

func TestSplitterWriterSplitSuccess(t *testing.T) {
	innerconn := &internal.FakeConn{}
	conn := internal.SplitterWriter{
		Conn: innerconn,
		Splitter: func(b []byte) [][]byte {
			return [][]byte{
				b[:2], b[2:],
			}
		},
	}
	const data = "deadbeef"
	count, err := conn.Write([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if count != len(data) {
		t.Fatal("invalid count")
	}
	if len(innerconn.WriteData) != 2 {
		t.Fatal("invalid data queue")
	}
	if string(innerconn.WriteData[0]) != "de" {
		t.Fatal("invalid written data[0]")
	}
	if string(innerconn.WriteData[1]) != "adbeef" {
		t.Fatal("invalid written data[1]")
	}
}

func TestSplitterWriterSplitFailure(t *testing.T) {
	expected := errors.New("mocked error")
	innerconn := &internal.FakeConn{WriteError: expected}
	conn := internal.SplitterWriter{
		Conn: innerconn,
		Splitter: func(b []byte) [][]byte {
			return [][]byte{
				b[:2], b[2:],
			}
		},
	}
	const data = "deadbeef"
	count, err := conn.Write([]byte(data))
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if count != 0 {
		t.Fatal("invalid count")
	}
	if len(innerconn.WriteData) != 0 {
		t.Fatal("invalid data queue")
	}
}

func TestWritevWorksWithAlsoEmptyData(t *testing.T) {
	conn := &internal.FakeConn{}
	datalist := [][]byte{
		[]byte("deadbeef"),
		[]byte(""),
		[]byte("dead"),
		nil,
		[]byte("badidea"),
		nil,
	}
	count, err := internal.Writev(conn, datalist)
	if err != nil {
		t.Fatal(err)
	}
	if count != 19 {
		t.Fatal("invalid number of bytes written")
	}
}

func TestWritevFailsAsIntended(t *testing.T) {
	expected := errors.New("mocked error")
	conn := &internal.FakeConn{WriteError: expected}
	datalist := [][]byte{
		[]byte("deadbeef"),
		[]byte(""),
		[]byte("dead"),
		nil,
		[]byte("badidea"),
		nil,
	}
	count, err := internal.Writev(conn, datalist)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if count != 0 {
		t.Fatal("invalid number of bytes written")
	}
}
