package must

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCreateFile(t *testing.T) {
	filename := filepath.Join("testdata", "test.txt")
	filep := CreateFile(filename)
	if _, err := filep.WriteString("antani"); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filename)
	filep.MustClose()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "antani" {
		t.Fatal("did not write the expected content")
	}
}

func TestOpenFile(t *testing.T) {
	filename := filepath.Join("testdata", ".gitignore")
	filep := OpenFile(filename)
	data, err := io.ReadAll(filep)
	if err != nil {
		t.Fatal(err)
	}
	filep.MustClose()
	if string(data) != "*\n" && string(data) != "*\r\n" {
		t.Fatal("unexpected content")
	}
}

func TestFprintf(t *testing.T) {
	w := &bytes.Buffer{}
	Fprintf(w, "hello %s", "world")
	if w.String() != "hello world" {
		t.Fatal("unexpected buffer content")
	}
}

func TestParseURL(t *testing.T) {
	URL := ParseURL("https://www.google.com/")
	if URL.Scheme != "https" || URL.Host != "www.google.com" || URL.Path != "/" {
		t.Fatal("unexpected parsed URL")
	}
}

func TestMarshalJSON(t *testing.T) {
	data := MarshalJSON("foobar")
	if string(data) != "\"foobar\"" {
		t.Fatal("incorrect marshalling")
	}
}

type example struct {
	Name string
	Age  int
}

func TestMarshalAndIndentJSON(t *testing.T) {
	input := &example{Name: "sbs", Age: 40}
	data := MarshalAndIndentJSON(input, "", "    ")
	expected := []byte("{\n    \"Name\": \"sbs\",\n    \"Age\": 40\n}")
	if diff := cmp.Diff(expected, data); diff != "" {
		t.Fatal(diff)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	input := []byte("{\n    \"Name\": \"sbs\",\n    \"Age\": 40\n}")
	var entry example
	UnmarshalJSON(input, &entry)
	if entry.Name != "sbs" || entry.Age != 40 {
		t.Fatal("did not unmarshal correctly")
	}
}

func TestListen(t *testing.T) {
	conn := Listen("tcp", "127.0.0.1:0")
	// TODO(bassosimone): unclear to me what to test here?
	conn.Close()
}

func TestNewHTTPRequest(t *testing.T) {
	req := NewHTTPRequest("GET", "https://www.google.com/", nil)
	if req.Method != "GET" {
		t.Fatal("invalid method")
	}
	URL := req.URL
	if URL.Scheme != "https" || URL.Host != "www.google.com" || URL.Path != "/" {
		t.Fatal("unexpected parsed URL")
	}
}

func TestSplitHostPort(t *testing.T) {
	addr, port := SplitHostPort("127.0.0.1:8080")
	if addr != "127.0.0.1" || port != "8080" {
		t.Fatal("unexpected result")
	}
}
