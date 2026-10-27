package quicfp

import "testing"

func TestParseWireClientHelloRecoversSNI(t *testing.T) {
	ch, err := buildClientHello("classifier-test.example")
	if err != nil {
		t.Fatal(err)
	}
	_, sni, err := parseWireClientHello(ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if sni != "classifier-test.example" {
		t.Fatalf("got SNI %q, want %q", sni, "classifier-test.example")
	}
}

func TestParseWireClientHelloUnmangledMatchesEverywhere(t *testing.T) {
	ch, err := buildClientHello("example.com")
	if err != nil {
		t.Fatal(err)
	}
	fields, _, err := parseWireClientHello(ch.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 8 {
		t.Fatalf("expected 8 fields, got %d", len(fields))
	}
	for _, f := range fields {
		if f.Declared != f.Correct {
			t.Errorf("field %s: unmangled ClientHello should have Declared==Correct, got %d vs %d", f.Name, f.Declared, f.Correct)
		}
		if _, ok := classifyLengthPermutation(f); ok {
			t.Errorf("field %s: classifyLengthPermutation should report no manipulation on a clean ClientHello", f.Name)
		}
	}
}

// TestClassifierRecoversEveryRealVectorPermutation builds every TLS-length
// vector quicfp itself produces (via the same code path vectors.go uses),
// parses the resulting wire bytes back with parseWireClientHello, and
// checks classifyLengthPermutation names the correct field and permutation
// -- the exact classification the mock censor's DPI logic depends on.
func TestClassifierRecoversEveryRealVectorPermutation(t *testing.T) {
	for _, spec := range tlsLengthFieldSpecs() {
		spec := spec
		for _, perm := range spec.Permutations {
			perm := perm
			t.Run(spec.FieldName+"/"+perm, func(t *testing.T) {
				ch, err := buildClientHello("example.com")
				if err != nil {
					t.Fatal(err)
				}
				patched := applyPermutation(ch, spec.Select(ch), perm)

				fields, sni, err := parseWireClientHello(patched)
				if err != nil {
					t.Fatalf("parseWireClientHello failed: %v", err)
				}
				if sni != "example.com" {
					t.Fatalf("SNI mismatch: got %q", sni)
				}

				var gotField, gotPerm string
				matches := 0
				for _, f := range fields {
					if p, ok := classifyLengthPermutation(f); ok {
						gotField, gotPerm = f.Name, p
						matches++
					}
				}
				if matches != 1 {
					t.Fatalf("expected exactly one field to classify as manipulated, got %d (field=%s perm=%s)", matches, gotField, gotPerm)
				}
				if gotField != spec.FieldName {
					t.Errorf("classified field = %s, want %s", gotField, spec.FieldName)
				}
				wantPerm := perm
				if spec.FieldName == "compression_methods_length" && perm == PermHalf {
					// Genuine wire-level ambiguity, not a classifier bug:
					// this field's Correct value is 1, so Half (1/2,
					// integer division) is bit-identical to Zero. A real
					// DPI parsing the field cannot tell them apart either.
					wantPerm = PermZero
				}
				if gotPerm != wantPerm {
					t.Errorf("classified permutation = %s, want %s", gotPerm, wantPerm)
				}
			})
		}
	}
}
