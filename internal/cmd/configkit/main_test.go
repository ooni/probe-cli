package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMapKind(t *testing.T) {
	ok := map[reflect.Kind]string{
		reflect.String: "string",
		reflect.Int64:  "integer",
		reflect.Int:    "integer",
		reflect.Int32:  "integer",
		reflect.Bool:   "boolean",
	}
	for k, want := range ok {
		got, err := mapKind(k)
		if err != nil {
			t.Errorf("mapKind(%v) unexpected error: %v", k, err)
		}
		if got != want {
			t.Errorf("mapKind(%v) = %q, want %q", k, got, want)
		}
	}
	for _, k := range []reflect.Kind{reflect.Float64, reflect.Slice, reflect.Map, reflect.Uint64} {
		if _, err := mapKind(k); err == nil {
			t.Errorf("mapKind(%v): expected error, got nil", k)
		}
	}
}

func TestJSONKey(t *testing.T) {
	type tagTest struct {
		A string `json:"a_key,omitempty"`
		B string `json:"b_key"`
		C string
		D string `json:",omitempty"`
	}
	typ := reflect.TypeOf(tagTest{})
	want := map[string]string{"A": "a_key", "B": "b_key", "C": "C", "D": "D"}
	for field, exp := range want {
		f, _ := typ.FieldByName(field)
		if got := jsonKey(f); got != exp {
			t.Errorf("jsonKey(%s) = %q, want %q", field, got, exp)
		}
	}
}

// EmbedPart is an exported embedded struct.
type EmbedPart struct {
	EmbeddedField string `json:"embedded_field,omitempty" ooni:"must not appear"`
}

type testConfig struct {
	EmbedPart
	Tagged   string `json:"tagged_key,omitempty" ooni:"a tagged option"`
	NoJSON   bool   `ooni:"option without a json tag"`
	NoOoni   string `json:"no_ooni"`
	skipped  string `ooni:"unexported"`
	PlainInt int64  `json:"count,omitempty" ooni:"a count"`
}

func TestSchemaProperties(t *testing.T) {
	tests := []struct {
		name    string
		input   reflect.Type
		want    []property
		wantErr bool
	}{{
		name:  "filters, orders, and keys fields correctly",
		input: reflect.TypeOf(testConfig{}),
		want: []property{
			{key: "tagged_key", typ: "string", desc: "a tagged option"},
			{key: "NoJSON", typ: "boolean", desc: "option without a json tag"},
			{key: "count", typ: "integer", desc: "a count"},
		},
	}, {
		name: "errors on an unsupported ooni-tagged field type",
		input: reflect.TypeOf(struct {
			Bad float64 `ooni:"unsupported type"`
		}{}),
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := schemaProperties(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("schemaProperties mismatch:\n got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

func TestRenderSchema(t *testing.T) {
	tests := []struct {
		name       string
		expName    string
		expVersion string
		props      []property
		contains   []string
	}{{
		name:       "with properties",
		expName:    "example",
		expVersion: "0.1.0",
		props:      []property{{key: "foo", typ: "string", desc: "a foo"}},
		contains: []string{
			`"$schema"`,
			`"experiment_name": "example"`,
			`"example experiment config"`,
			`"experiment_version": "0.1.0"`,
			`"foo"`,
			`"a foo"`,
		},
	}, {
		name:       "empty properties",
		expName:    "dash",
		expVersion: "0.0.0",
		props:      nil,
		contains: []string{
			`"experiment_name": "dash"`,
			`"properties": {}`,
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderSchema(tt.expName, tt.expVersion, tt.props)
			if err != nil {
				t.Fatal(err)
			}
			s := string(out)

			// it must always be valid JSON
			var parsed map[string]any
			if err := json.Unmarshal(out, &parsed); err != nil {
				t.Fatalf("output is not valid JSON: %v\n%s", err, s)
			}

			for _, want := range tt.contains {
				if !strings.Contains(s, want) {
					t.Errorf("missing %q in:\n%s", want, s)
				}
			}

			// key order is invariant: experiment_name < version < experiment_version < type < properties
			iname := strings.Index(s, `"experiment_name"`)
			iv := strings.Index(s, `"version"`)
			ie := strings.Index(s, `"experiment_version"`)
			it := strings.Index(s, `"type"`)
			ip := strings.Index(s, `"properties"`)
			if !(iname < iv && iv < ie && ie < it && it < ip) {
				t.Errorf("unexpected key order: experiment_name=%d version=%d experiment_version=%d type=%d properties=%d\n%s",
					iname, iv, ie, it, ip, s)
			}
		})
	}
}

func TestEmit(t *testing.T) {
	dir := t.TempDir()
	want := []byte("{}\n")

	// write mode creates <base>/<sub>/config.json
	if d, err := emit(dir, "example", defaultSubdir, want, false); err != nil || d != 0 {
		t.Fatalf("emit write: drift=%d err=%v", d, err)
	}
	path := filepath.Join(dir, "example", defaultSubdir, "config.json")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("emit wrote %q, want %q", got, want)
	}

	if d, err := emit(dir, "example", defaultSubdir, want, true); err != nil || d != 0 {
		t.Fatalf("emit check in-sync: drift=%d err=%v", d, err)
	}
	if d, _ := emit(dir, "example", defaultSubdir, []byte("different\n"), true); d != 1 {
		t.Fatalf("emit check stale: drift=%d, want 1", d)
	}
	if d, _ := emit(dir, "does_not_exist", "v9.9", want, true); d != 1 {
		t.Fatalf("emit check missing: drift=%d, want 1", d)
	}
}

func TestRunGeneratesAndChecks(t *testing.T) {
	dir := t.TempDir()

	drift, err := run(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if drift != 0 {
		t.Fatalf("generate reported drift=%d", drift)
	}

	// freshly generated output must be in sync with itself
	drift, err = run(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if drift != 0 {
		t.Fatalf("check after generate reported drift=%d, want 0", drift)
	}

	exampleDefault := filepath.Join(dir, "example", "default", "config.json")
	exampleVersion := filepath.Join(dir, "example", "v0.1", "config.json")
	data, err := os.ReadFile(exampleDefault)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := os.ReadFile(exampleVersion)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, versioned) {
		t.Fatal("example default/ and v0.1/ schemas differ")
	}
	for _, want := range []string{`"experiment_name": "example"`, `"message"`, `"return_error"`, `"experiment_version"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("example schema missing %q:\n%s", want, data)
		}
	}

	// a multi-version experiment emits default/ plus one dir per version
	for _, sub := range []string{"default", "v0.4", "v0.5"} {
		p := filepath.Join(dir, "web_connectivity", sub, "config.json")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing web_connectivity schema %s: %v", sub, err)
		}
	}
	// web_connectivity's default is v0.4, so default/ must equal v0.4/
	wcDefault, err := os.ReadFile(filepath.Join(dir, "web_connectivity", "default", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	wcV04, err := os.ReadFile(filepath.Join(dir, "web_connectivity", "v0.4", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wcDefault, wcV04) {
		t.Fatal("web_connectivity default/ must equal v0.4/ (its default version)")
	}

	// tampering must be detected
	if err := os.WriteFile(exampleDefault, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	drift, err = run(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if drift == 0 {
		t.Fatal("expected drift after tampering with a schema, got 0")
	}
}
