package oonimkall_test

import (
	"encoding/json"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestXOONIRun(t *testing.T) {
	t.Run("smoke testing of the fetch API", func(t *testing.T) {
		sess, err := NewSessionForTesting()
		if err != nil {
			t.Fatal(err)
		}

		resp, err := sess.OONIRunFetch(sess.NewContext(), 297500125102)
		if err != nil {
			t.Fatal(err)
		}

		data := runtimex.Try1(json.Marshal(resp))
		t.Log(string(data))
	})

}
