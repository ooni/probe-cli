package oonimkall_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestXOONIRun(t *testing.T) {
	t.Run("smoke testing of the fetch API", func(t *testing.T) {
		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		rawResp, err := sess.OONIRunFetch(sess.NewContext(), 297500125102)
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]any{
			"creation_time": "2023-06-06T09:19:41Z",
			"descriptor": map[string]any{
				"author":           "ooni",
				"description":      "Check whether [WhatsApp](https://ooni.org/nettest/whatsapp/) is blocked",
				"description_intl": map[string]any{},
				"icon":             "Md123",
				"name":             "Updated 2 Instant Messaging",
				"name_intl": map[string]any{
					"it": "Instant Messaging IT",
				},
				"nettests": []any{
					map[string]any{
						"backend_options":           map[string]any{},
						"inputs":                    []any{},
						"is_background_run_enabled": true,
						"is_manual_run_enabled":     false,
						"options":                   map[string]any{},
						"test_name":                 "whatsapp",
					},
				},
				"short_description":      "Test the blocking of instant messaging apps",
				"short_description_intl": map[string]any{},
			},
			"v": 1.0,
		}

		var got map[string]any
		runtimex.Try0(json.Unmarshal([]byte(rawResp), &got))
		t.Log(got)

		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})
}
