package httpapi

import (
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestDescriptor_WithBodyLogging(t *testing.T) {
	type fields struct {
		Accept        string
		Authorization string
		ContentType   string
		LogBody       bool
		MaxBodySize   int64
		Method        string
		RequestBody   []byte
		Timeout       time.Duration
		URLPath       string
		URLQuery      url.Values
	}
	tests := []struct {
		name   string
		fields fields
		want   *Descriptor
	}{{
		name:   "with empty fields",
		fields: fields{}, // LogBody defaults to false
		want: &Descriptor{
			LogBody: true,
		},
	}, {
		name: "with nonempty fields",
		fields: fields{
			Accept:        "xx",
			Authorization: "y",
			ContentType:   "zzz",
			LogBody:       false, // obviously must be false
			MaxBodySize:   123,
			Method:        "POST",
			RequestBody:   []byte("123"),
			Timeout:       15555,
			URLPath:       "/",
			URLQuery: map[string][]string{
				"a": {"b"},
			},
		},
		want: &Descriptor{
			Accept:        "xx",
			Authorization: "y",
			ContentType:   "zzz",
			LogBody:       true,
			MaxBodySize:   123,
			Method:        "POST",
			RequestBody:   []byte("123"),
			Timeout:       15555,
			URLPath:       "/",
			URLQuery: map[string][]string{
				"a": {"b"},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := &Descriptor{
				Accept:        tt.fields.Accept,
				Authorization: tt.fields.Authorization,
				ContentType:   tt.fields.ContentType,
				LogBody:       tt.fields.LogBody,
				MaxBodySize:   tt.fields.MaxBodySize,
				Method:        tt.fields.Method,
				RequestBody:   tt.fields.RequestBody,
				Timeout:       tt.fields.Timeout,
				URLPath:       tt.fields.URLPath,
				URLQuery:      tt.fields.URLQuery,
			}
			got := desc.WithBodyLogging(true)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewGetJSONDescriptor(t *testing.T) {
	expected := &Descriptor{
		Accept:        "application/json",
		Authorization: "",
		ContentType:   "",
		LogBody:       false,
		MaxBodySize:   DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil,
		Timeout:       DefaultCallTimeout,
		URLPath:       "/robots.txt",
		URLQuery:      url.Values{},
	}
	got := NewGETJSONDescriptor("/robots.txt")
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewGetJSONWithQueryDescriptor(t *testing.T) {
	query := url.Values{
		"a": {"b"},
		"c": {"d"},
	}
	expected := &Descriptor{
		Accept:        "application/json",
		Authorization: "",
		ContentType:   "",
		LogBody:       false,
		MaxBodySize:   DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil,
		Timeout:       DefaultCallTimeout,
		URLPath:       "/robots.txt",
		URLQuery:      query,
	}
	got := NewGETJSONWithQueryDescriptor("/robots.txt", query)
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewPOSTJSONWithJSONResponseDescriptor(t *testing.T) {
	type request struct {
		Name string
		Age  int64
	}

	t.Run("with failure", func(t *testing.T) {
		request := make(chan int64)
		got, err := NewPOSTJSONWithJSONResponseDescriptor("/robots.txt", request)
		if err == nil || err.Error() != "json: unsupported type: chan int64" {
			log.Fatal("unexpected err", err)
		}
		if got != nil {
			log.Fatal("expected to get a nil Descriptor")
		}
	})

	t.Run("with success", func(t *testing.T) {
		request := request{
			Name: "sbs",
			Age:  99,
		}
		expected := &Descriptor{
			Accept:        "application/json",
			Authorization: "",
			ContentType:   "application/json",
			LogBody:       false,
			MaxBodySize:   DefaultMaxBodySize,
			Method:        http.MethodPost,
			RequestBody:   []byte(`{"Name":"sbs","Age":99}`),
			Timeout:       DefaultCallTimeout,
			URLPath:       "/robots.txt",
			URLQuery:      nil,
		}
		got, err := NewPOSTJSONWithJSONResponseDescriptor("/robots.txt", request)
		if err != nil {
			log.Fatal(err)
		}
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestMustNewPOSTJSONWithJSONResponseDescriptor(t *testing.T) {
	type request struct {
		Name string
		Age  int64
	}

	t.Run("with failure", func(t *testing.T) {
		var panicked bool
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			request := make(chan int64)
			_ = MustNewPOSTJSONWithJSONResponseDescriptor("/robots.txt", request)
		}()
		if !panicked {
			t.Fatal("did not panic")
		}
	})

	t.Run("with success", func(t *testing.T) {
		request := request{
			Name: "sbs",
			Age:  99,
		}
		expected := &Descriptor{
			Accept:        "application/json",
			Authorization: "",
			ContentType:   "application/json",
			LogBody:       false,
			MaxBodySize:   DefaultMaxBodySize,
			Method:        http.MethodPost,
			RequestBody:   []byte(`{"Name":"sbs","Age":99}`),
			Timeout:       DefaultCallTimeout,
			URLPath:       "/robots.txt",
			URLQuery:      nil,
		}
		got := MustNewPOSTJSONWithJSONResponseDescriptor("/robots.txt", request)
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewGetResourceDescriptor(t *testing.T) {
	expected := &Descriptor{
		Accept:        "",
		Authorization: "",
		ContentType:   "",
		LogBody:       false,
		MaxBodySize:   DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil,
		Timeout:       DefaultCallTimeout,
		URLPath:       "/robots.txt",
		URLQuery:      url.Values{},
	}
	got := NewGETResourceDescriptor("/robots.txt")
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Fatal(diff)
	}
}
