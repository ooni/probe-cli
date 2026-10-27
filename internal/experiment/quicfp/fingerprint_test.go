package quicfp

import "testing"

func makePing(blocked bool) *SinglePing {
	return &SinglePing{Blocked: blocked}
}

func TestAggregateOutcome(t *testing.T) {
	cases := []struct {
		name   string
		pings  []*SinglePing
		expect Outcome
	}{
		{"all blocked", []*SinglePing{makePing(true), makePing(true), makePing(true)}, OutcomeBlocked},
		{"all bypassed", []*SinglePing{makePing(false), makePing(false), makePing(false)}, OutcomeBypassed},
		{"two of three blocked hits threshold", []*SinglePing{makePing(true), makePing(true), makePing(false)}, OutcomeBlocked},
		{"one of three blocked stays bypassed", []*SinglePing{makePing(true), makePing(false), makePing(false)}, OutcomeBypassed},
		{"even split is inconsistent", []*SinglePing{makePing(true), makePing(false)}, OutcomeInconsistent},
		{"no pings is unknown", nil, OutcomeUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AggregateOutcome(c.pings, DefaultAgreementThreshold)
			if got != c.expect {
				t.Errorf("got %s want %s", got, c.expect)
			}
		})
	}
}

func TestAggregateOutcomeIgnoresBuildErrors(t *testing.T) {
	failure := "boom"
	buildErrPing := &SinglePing{Failure: &failure, Blocked: false} // build/send error, not a network signal
	pings := []*SinglePing{buildErrPing, makePing(false), makePing(false)}
	got := AggregateOutcome(pings, DefaultAgreementThreshold)
	if got != OutcomeBypassed {
		t.Fatalf("build errors should be excluded from the denominator, got %s", got)
	}
}

func TestOutcomeJSONRoundTrip(t *testing.T) {
	for _, o := range []Outcome{OutcomeUnknown, OutcomeBypassed, OutcomeBlocked, OutcomeInconsistent} {
		b, err := o.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var got Outcome
		if err := got.UnmarshalJSON(b); err != nil {
			t.Fatal(err)
		}
		if got != o {
			t.Errorf("round trip failed for %s: got %s", o, got)
		}
	}
}

func TestMatchSignaturePerfectMatch(t *testing.T) {
	observed := map[string]Outcome{
		"tls_length_message_length_zero": OutcomeBlocked,
		"tls_length_message_length_max":  OutcomeBypassed,
		"quic_version_ietf_v2":           OutcomeBlocked,
	}
	sig := Signature{
		Device: "synthetic-device-a",
		Vectors: map[string]Outcome{
			"tls_length_message_length_zero": OutcomeBlocked,
			"tls_length_message_length_max":  OutcomeBypassed,
			"quic_version_ietf_v2":           OutcomeBlocked,
		},
	}
	res := MatchSignature(observed, sig, Weights{}, len(observed))
	if res.Score != 1.0 {
		t.Fatalf("expected perfect score, got %f", res.Score)
	}
	if res.Compared != 3 || res.Agreements != 3 || res.Disagreements != 0 {
		t.Fatalf("unexpected counts: %+v", res)
	}
}

func TestMatchSignatureIgnoresUnknownAndInconsistent(t *testing.T) {
	observed := map[string]Outcome{
		"a": OutcomeBlocked,
		"b": OutcomeInconsistent, // must not count for or against any signature
		"c": OutcomeUnknown,      // ditto
	}
	sig := Signature{
		Device: "synthetic-device-b",
		Vectors: map[string]Outcome{
			"a": OutcomeBlocked,
			"b": OutcomeBypassed, // would disagree if it were compared
			"c": OutcomeBlocked,  // would disagree if it were compared
			"d": OutcomeBlocked,  // not observed at all
		},
	}
	res := MatchSignature(observed, sig, Weights{}, len(observed))
	if res.Compared != 1 || res.Agreements != 1 {
		t.Fatalf("expected exactly one comparable vector, got %+v", res)
	}
	if res.Score != 1.0 {
		t.Fatalf("expected score 1.0 from the single agreeing vector, got %f", res.Score)
	}
}

func TestRankSignaturesOrdersByScoreThenSupport(t *testing.T) {
	observed := map[string]Outcome{
		"a": OutcomeBlocked,
		"b": OutcomeBlocked,
		"c": OutcomeBypassed,
	}
	sigs := []Signature{
		{
			Device: "weak-support-perfect-score",
			Vectors: map[string]Outcome{
				"a": OutcomeBlocked,
			},
		},
		{
			Device: "strong-support-perfect-score",
			Vectors: map[string]Outcome{
				"a": OutcomeBlocked,
				"b": OutcomeBlocked,
				"c": OutcomeBypassed,
			},
		},
		{
			Device: "disagrees-everywhere",
			Vectors: map[string]Outcome{
				"a": OutcomeBypassed,
				"b": OutcomeBypassed,
				"c": OutcomeBlocked,
			},
		},
	}
	ranked := RankSignatures(observed, sigs, nil)
	if ranked[0].Device != "strong-support-perfect-score" {
		t.Fatalf("expected the more-supported perfect match first, got %+v", ranked)
	}
	if ranked[len(ranked)-1].Device != "disagrees-everywhere" {
		t.Fatalf("expected the total-disagreement signature last, got %+v", ranked)
	}
}

func TestDeriveWeightsZeroesUnanimousVectorsAndFavorsEvenSplits(t *testing.T) {
	sigs := []Signature{
		{Device: "_meta_entry_should_be_ignored", Vectors: map[string]Outcome{
			"unanimous_blocked": OutcomeBypassed, // would corrupt the tally if not skipped
		}},
		{Device: "d1", Vectors: map[string]Outcome{
			"unanimous_blocked": OutcomeBlocked,
			"even_split":        OutcomeBlocked,
		}},
		{Device: "d2", Vectors: map[string]Outcome{
			"unanimous_blocked": OutcomeBlocked,
			"even_split":        OutcomeBypassed,
		}},
	}
	w := DeriveWeights(sigs)
	if got := w.get("unanimous_blocked"); got != 0 {
		// get() treats an explicit 0 the same as "absent" (defaults to
		// 1.0), so check the map directly instead.
		if v, ok := w["unanimous_blocked"]; !ok || v != 0 {
			t.Fatalf("expected unanimous_blocked to have weight 0, map has %v (ok=%v)", v, ok)
		}
	}
	if got := w["even_split"]; got != 1.0 {
		t.Fatalf("expected even_split (50/50 across devices) to have weight 1.0, got %f", got)
	}
}

func TestWeightsDefaultsToOne(t *testing.T) {
	var w Weights
	if got := w.get("anything"); got != 1.0 {
		t.Fatalf("nil Weights should default every vector to weight 1.0, got %f", got)
	}
}
