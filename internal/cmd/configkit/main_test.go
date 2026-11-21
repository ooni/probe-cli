package main

import (
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
	got, err := schemaProperties(reflect.TypeOf(testConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []property{
		{key: "tagged_key", typ: "string", desc: "a tagged option"},
		{key: "NoJSON", typ: "boolean", desc: "option without a json tag"},
		{key: "count", typ: "integer", desc: "a count"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("schemaProperties mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestSchemaPropertiesUnsupportedType(t *testing.T) {
	type badConfig struct {
		Bad float64 `ooni:"unsupported type"`
	}
	if _, err := schemaProperties(reflect.TypeOf(badConfig{})); err == nil {
		t.Fatal("expected error for an unsupported ooni-tagged field type, got nil")
	}
}

func TestRenderSchema(t *testing.T) {
	out, err := renderSchema("example", "0.1.0", []property{
		{key: "foo", typ: "string", desc: "a foo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, s)
	}

	for _, want := range []string{
		`"$schema"`,
		`"example experiment config"`,
		`"version"`,
		`"experiment_version": "0.1.0"`,
		`"foo"`,
		`"a foo"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}

	iv := strings.Index(s, `"version"`)
	ie := strings.Index(s, `"experiment_version"`)
	it := strings.Index(s, `"type"`)
	ip := strings.Index(s, `"properties"`)
	if !(iv < ie && ie < it && it < ip) {
		t.Errorf("unexpected key order: version=%d experiment_version=%d type=%d properties=%d\n%s",
			iv, ie, it, ip, s)
	}
}

func TestRenderSchemaEmptyProperties(t *testing.T) {
	out, err := renderSchema("dash", "0.0.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"properties": {}`) {
		t.Errorf("expected empty properties object, got:\n%s", out)
	}
}

func TestSchemaPath(t *testing.T) {
	cases := map[string]string{
		"example":               filepath.Join("schemas", "example", "config.json"),
		"web_connectivity@v0.5": filepath.Join("schemas", "web_connectivity", "v0.5", "config.json"),
	}
	for name, want := range cases {
		if got := schemaPath("schemas", name); got != want {
			t.Errorf("schemaPath(%q) = %q, want %q", name, got, want)
		}
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

	// spot-check a known experiment
	examplePath := filepath.Join(dir, "example", "config.json")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"message"`, `"return_error"`, `"experiment_version"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("example schema missing %q:\n%s", want, data)
		}
	}

	// tampering must be detected
	if err := os.WriteFile(examplePath, []byte("{}\n"), 0644); err != nil {
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
