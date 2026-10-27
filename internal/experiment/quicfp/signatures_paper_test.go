package quicfp

import "testing"

// knownUncertainVectors are the vectors table9_from_paper.json intentionally
// leaves as "unknown" -- see that file's _caveats entry for why (currently
// just the one vector with no corresponding row in either published table).
func knownUncertainVectors() map[string]bool {
	return map[string]bool{
		"quic_feature_oversized_datagram": true,
	}
}

func TestPaperSignaturesMatchRealVectorNamesAndArePopulated(t *testing.T) {
	sigs, err := LoadSignatures("testdata/table9_from_paper.json")
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
	uncertain := knownUncertainVectors()

	for _, sig := range sigs {
		if isMetaSignature(sig) {
			continue
		}
		for name, outcome := range sig.Vectors {
			if !realNames[name] {
				t.Errorf("device %q references unknown vector %q", sig.Device, name)
			}
			if outcome == OutcomeUnknown && !uncertain[name] {
				t.Errorf("device %q vector %q left as unknown but isn't in knownUncertainVectors -- "+
					"either populate it or add it there with a reason", sig.Device, name)
			}
			if outcome != OutcomeUnknown && uncertain[name] {
				t.Errorf("device %q vector %q is populated (%s) but knownUncertainVectors says it "+
					"shouldn't be trusted yet -- update knownUncertainVectors if this was "+
					"deliberately verified", sig.Device, name, outcome)
			}
		}
		for name := range realNames {
			if _, ok := sig.Vectors[name]; !ok {
				t.Errorf("device %q is missing vector %q present in the code", sig.Device, name)
			}
		}
	}
}

// TestFingerprintRankerRecoversEachDeviceFromItself feeds each real device's
// own Vectors back into RankSignatures as the "observed" data and checks
// the ranker recovers that same device at score 1.0, first place. This is
// the end-to-end validation that both the transcription and the scoring
// engine agree with each other on real data, not just synthetic examples.
func TestFingerprintRankerRecoversEachDeviceFromItself(t *testing.T) {
	sigs, err := LoadSignatures("testdata/table9_from_paper.json")
	if err != nil {
		t.Fatal(err)
	}
	weights := DeriveWeights(sigs)

	for _, sig := range sigs {
		if isMetaSignature(sig) {
			continue
		}
		t.Run(sig.Device, func(t *testing.T) {
			ranked := RankSignatures(sig.Vectors, sigs, weights)
			if len(ranked) == 0 {
				t.Fatal("no ranked results")
			}
			top := ranked[0]
			if top.Device != sig.Device {
				t.Fatalf("expected %s to rank itself first, got %s (score=%f)", sig.Device, top.Device, top.Score)
			}
			if top.Score != 1.0 {
				t.Fatalf("expected a perfect self-match score, got %f", top.Score)
			}
		})
	}
}

// physicalTable12Rows maps each of Table 12's 34 physical QUIC rows to one
// representative quicfp vector name. Several quicfp vectors expand a single
// combined Table 12 row (e.g. "Cipher Suite Length: Zero/Half/0xFFFF") into
// multiple identically-valued vectors; counting every expanded vector would
// inflate a "differs from TSG" tally relative to what Table 12 itself
// reports, so this list picks exactly one vector per real row.
func physicalTable12Rows() []string {
	return []string{
		"tls_length_message_length_zero", "tls_length_message_length_double",
		"tls_length_message_length_half", "tls_length_message_length_max",
		"tls_length_message_length_exclude_last_extension",
		"tls_length_session_id_length_too_long",
		"tls_length_cipher_suite_length_double",
		"tls_length_cipher_suite_length_zero",          // represents Zero/Half/0xFFFF
		"tls_length_compression_methods_length_zero",   // represents Zero/Half
		"tls_length_compression_methods_length_double", // represents Double/0xFF
		"tls_length_extensions_length_zero", "tls_length_extensions_length_half",
		"tls_length_extensions_length_double", // represents Double/0xFFFF
		"tls_length_extensions_length_exclude_last_extension",
		"tls_length_sni_extension_length_zero", // represents Zero/Half
		"tls_length_sni_extension_length_double", "tls_length_sni_extension_length_max",
		"tls_length_sni_list_length_zero", "tls_length_sni_list_length_half",
		"tls_length_sni_list_length_double", "tls_length_sni_list_length_max",
		"tls_length_sni_name_length_zero",   // represents Zero/Half
		"tls_length_sni_name_length_double", // represents Double/0xFFFF
		"quic_version_gquic_q023_q043", "quic_version_gquic_q041_q045",
		"quic_version_gquic_q046_q049", "quic_version_gquic_t050_t051",
		"quic_version_ietf_draft22_draft28", "quic_version_ietf_draft29_v1",
		"quic_version_ietf_v2",
		"quic_feature_fragmentation", "quic_feature_below_1200_bytes",
		"quic_feature_forced_negotiation", "quic_feature_multi_packet_initial",
	}
}

// TestVectorsDifferentToTSGMatchesPaper counts, per QUIC column, the
// physical Table 12 rows that differ from TSG, and compares them with the
// table's own "# vectors different to TSG" row. That row prints KZ=8, but
// the table's KZ cells differ from TSG in 9 rows (Message Length 0xFFFF,
// Extensions Length Double/0xFFFF, SNI List Length Double and 0xFFFF, the
// Q041 and Q046 groups, T050, draft-22-28 and Below 1200 Bytes), so this test
// follows the cells: MM=11, PK=12 and GFW=8 match the printed row.
func TestVectorsDifferentToTSGMatchesPaper(t *testing.T) {
	sigs, err := LoadSignatures("testdata/table9_from_paper.json")
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Signature{}
	for _, s := range sigs {
		if !isMetaSignature(s) {
			by[s.Device] = s
		}
	}

	want := map[string]int{"GFW": 8, "Kazakhstan": 9, "Myanmar": 11, "Pakistan": 12}
	got := map[string]int{}
	tsg := by["TSG"].Vectors
	for _, name := range physicalTable12Rows() {
		tsgVal := tsg[name]
		for device := range want {
			if by[device].Vectors[name] != tsgVal {
				got[device]++
			}
		}
	}
	for device, wantCount := range want {
		if got[device] != wantCount {
			t.Errorf("%s: got %d vectors different from TSG, the table's cells give %d", device, got[device], wantCount)
		}
	}
}

func TestPaperSignaturesUnanimousRows(t *testing.T) {
	sigs, err := LoadSignatures("testdata/table9_from_paper.json")
	if err != nil {
		t.Fatal(err)
	}
	devices := map[string]Signature{}
	for _, s := range sigs {
		if !isMetaSignature(s) {
			devices[s.Device] = s
		}
	}
	if len(devices) != 5 {
		t.Fatalf("expected 5 populated devices, got %d", len(devices))
	}

	unanimous := map[string]Outcome{
		"quic_version_gquic_q023_q043":                        OutcomeBlocked,
		"quic_version_ietf_draft29_v1":                        OutcomeBlocked,
		"quic_feature_fragmentation":                          OutcomeBlocked,
		"quic_feature_forced_negotiation":                     OutcomeBypassed,
		"quic_feature_multi_packet_initial":                   OutcomeBypassed,
		"tls_length_extensions_length_half":                   OutcomeBlocked,
		"tls_length_extensions_length_exclude_last_extension": OutcomeBlocked,
		"tls_length_sni_extension_length_max":                 OutcomeBypassed,
		"tls_length_sni_name_length_zero":                     OutcomeBypassed,
		"tls_length_sni_name_length_half":                     OutcomeBypassed,
		"tls_length_sni_name_length_double":                   OutcomeBypassed,
		"tls_length_sni_name_length_max":                      OutcomeBypassed,
	}
	for vector, want := range unanimous {
		for device, sig := range devices {
			if got := sig.Vectors[vector]; got != want {
				t.Errorf("expected unanimous %s=%s but device %q has %s=%s", vector, want, device, vector, got)
			}
		}
	}
}
