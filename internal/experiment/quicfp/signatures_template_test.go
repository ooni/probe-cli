package quicfp

import "testing"

func TestSignatureTemplateMatchesRealVectorNames(t *testing.T) {
	sigs, err := LoadSignatures("testdata/table9_template.json")
	if err != nil {
		t.Fatal(err)
	}

	vectors, err := AllVectors("example.com")
	if err != nil {
		t.Fatal(err)
	}
	realNames := make(map[string]bool, len(vectors))
	for _, v := range vectors {
		realNames[v.Name] = true
	}

	for _, sig := range sigs {
		if sig.Device == "_instructions" {
			continue
		}
		for name, outcome := range sig.Vectors {
			if !realNames[name] {
				t.Errorf("device %q references unknown vector %q -- template has drifted from the code", sig.Device, name)
			}
			if outcome != OutcomeUnknown {
				t.Errorf("device %q vector %q is %q, not unknown -- this file should ship as a skeleton; "+
					"real values belong in your own copy, not the one shipped here", sig.Device, name, outcome)
			}
		}
		for name := range realNames {
			if _, ok := sig.Vectors[name]; !ok {
				t.Errorf("device %q is missing vector %q present in the code", sig.Device, name)
			}
		}
	}
}
