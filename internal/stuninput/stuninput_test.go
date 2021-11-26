package stuninput

import "testing"

func TestAsSnowflakeInput(t *testing.T) {
	outputs := AsSnowflakeInput()
	if len(outputs) != len(inputs) {
		t.Fatal("unexpected number of entries")
	}
	for idx := 0; idx < len(inputs); idx++ {
		output := outputs[idx]
		input := "stun:" + inputs[idx]
		if input != output {
			t.Fatal("mismatch")
		}
	}
}

func TestAsStunReachabilityInput(t *testing.T) {
	outputs := AsnStunReachabilityInput()
	if len(outputs) != len(inputs) {
		t.Fatal("unexpected number of entries")
	}
	for idx := 0; idx < len(inputs); idx++ {
		output := outputs[idx]
		input := "stun://" + inputs[idx]
		if input != output {
			t.Fatal("mismatch")
		}
	}
}
