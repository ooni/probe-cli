package stuninput

import (
	"strings"
	"testing"
)

func TestAsSnowflakeInput(t *testing.T) {
	outputs := AsSnowflakeInput()
	if len(outputs) != len(inputs) {
		t.Fatal("unexpected number of entries")
	}
	for _, output := range outputs {
		output = strings.TrimPrefix(output, "stun:")
		if !inputs[output] {
			t.Fatal("not found in inputs", output)
		}
	}
}

func TestAsStunReachabilityInput(t *testing.T) {
	outputs := AsnStunReachabilityInput()
	if len(outputs) != len(inputs) {
		t.Fatal("unexpected number of entries")
	}
	for _, output := range outputs {
		output = strings.TrimPrefix(output, "stun://")
		if !inputs[output] {
			t.Fatal("not found in inputs", output)
		}
	}
}
