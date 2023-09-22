package hujsonx

import (
	"errors"
	"io"
	"testing"
)

type user struct {
	Name string
	Age  int
}

func TestHuJSONXWorkingAsIntended(t *testing.T) {
	t.Run("for invalid input", func(t *testing.T) {
		input := []byte("{")
		var v user
		err := Unmarshal(input, &v)
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("for valid JSON we cannot map to a real struct", func(t *testing.T) {
		input := []byte(`{"Name": {}, "Age": []}`)
		var v user
		err := Unmarshal(input, &v)
		expected := "json: cannot unmarshal object into Go struct field user.Name of type string"
		if err == nil || err.Error() != expected {
			t.Fatal("unexpected error", err)
		}
	})
}
