package ooapi

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewDescriptorCheckIn(t *testing.T) {
	// Implementation note: this test uses reflection such that new
	// fields added to a Descriptor will cause an error if they aren't
	// initialized as expected (which may be zero-initialized).

	desc := NewDescriptorCheckIn(&model.OOAPICheckInConfig{})

	rdesc := reflect.ValueOf(desc).Elem()
	typ := rdesc.Type()
	for idx := 0; idx < rdesc.NumField(); idx++ {
		field := rdesc.Field(idx)
		name := typ.Field(idx).Name

		// check fields which should have a zero value first
		if field.IsZero() {
			switch name {
			case "Authorization", "LogBody", "MaxBodySize", "Timeout", "URLQuery":
				// this field is expected to be zero
			default:
				t.Fatalf("field %s should not be zero-initialized", name)
			}
			continue
		}

		// then focus on fields who should not have a zero value
		switch name {
		case "AcceptEncodingGzip":
			if !field.Interface().(bool) {
				t.Fatalf("unexpected desc.%s", name)
			}
		case "Accept":
			if field.Interface().(string) != httpapi.ApplicationJSON {
				t.Fatalf("unexpected desc.%s", name)
			}
		case "ContentType":
			if field.Interface().(string) != httpapi.ApplicationJSON {
				t.Fatalf("unexpected desc.%s", name)
			}
		case "Method":
			if field.Interface().(string) != http.MethodPost {
				t.Fatalf("unexpected desc.%s", name)
			}
		case "RequestBody":
			if len(field.Interface().([]byte)) <= 2 {
				t.Fatalf("unexpected desc.%s length", name)
			}
		case "URLPath":
			if field.Interface().(string) != "/api/v1/check-in" {
				t.Fatalf("unexpected desc.%s", name)
			}
		default:
			t.Fatalf("unhandled non-zero field %s", name)
		}
	}
}
