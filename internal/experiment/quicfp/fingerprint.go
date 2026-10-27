package quicfp

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

// isMetaSignature reports whether sig is a documentation/placeholder entry
// (by convention, Device starting with "_") rather than a real device, so
// callers can skip it without every signature file needing to filter these
// out by hand.
func isMetaSignature(sig Signature) bool {
	return strings.HasPrefix(sig.Device, "_")
}

// Signature is one known device's expected per-vector Outcome pattern, as
// reported for some prior measurement (ideally Table 9 itself, or your own
// earlier run against a device of known identity/version).
//
// IMPORTANT: this package does NOT ship Table 9's real values. The PDF this
// tool was built from lost the alignment between each row's checkmarks and
// their columns during text extraction, so any values transcribed from it
// would be a guess dressed up as ground truth. See
// testdata/table9_template.json for a ready-to-fill skeleton, and populate
// Vectors/Source yourself from the paper's Table 9 or its Zenodo artifact
// (referenced in the paper's Open Science section) before trusting
// MatchSignature's output.
type Signature struct {
	Device  string             `json:"device"`
	Source  string             `json:"source"`
	Notes   string             `json:"notes,omitempty"`
	Vectors map[string]Outcome `json:"vectors"`
}

// LoadSignatures reads a JSON array of Signature from path.
func LoadSignatures(path string) ([]Signature, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sigs []Signature
	if err := json.Unmarshal(b, &sigs); err != nil {
		return nil, err
	}
	return sigs, nil
}

// SaveSignatures writes sigs as indented JSON to path.
func SaveSignatures(path string, sigs []Signature) error {
	b, err := json.MarshalIndent(sigs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// DeriveWeights computes a discriminative weight per vector directly from
// a signature set, replacing guesswork like DefaultWeights with something
// computed from actual reference data. A vector every device agrees on
// (all blocked, or all bypassed) gets weight 0: it cannot help distinguish
// among *these* devices, though seeing a candidate disagree with that
// unanimity is still worth noting by hand (it may mean the candidate isn't
// any of the devices in your signature set at all -- see METHODOLOGY.md in
// the standalone quicfp repository). A vector split as evenly as possible
// across devices gets weight 1.0. Meta/placeholder signatures (see isMetaSignature) are
// ignored. Call this once you have a signature file loaded and prefer its
// output over DefaultWeights, which is only a fallback for when you don't.
func DeriveWeights(sigs []Signature) Weights {
	type counts struct{ blocked, bypassed int }
	tally := map[string]*counts{}
	for _, s := range sigs {
		if isMetaSignature(s) {
			continue
		}
		for name, o := range s.Vectors {
			c, ok := tally[name]
			if !ok {
				c = &counts{}
				tally[name] = c
			}
			switch o {
			case OutcomeBlocked:
				c.blocked++
			case OutcomeBypassed:
				c.bypassed++
			}
		}
	}
	w := make(Weights, len(tally))
	for name, c := range tally {
		total := c.blocked + c.bypassed
		if total == 0 {
			continue
		}
		p := float64(c.bypassed) / float64(total)
		w[name] = 4 * p * (1 - p) // Gini-style impurity, normalized to peak at 1.0 when p=0.5
	}
	return w
}

// Weights lets callers emphasize the vectors they trust most as
// discriminators. A weight of 0 or an absent entry defaults to 1.0.
// Prefer DeriveWeights(sigs), computed from real reference data, over
// hand-picking values here.
type Weights map[string]float64

func (w Weights) get(name string) float64 {
	if v, ok := w[name]; ok && v != 0 {
		return v
	}
	return 1.0
}

// DefaultWeights is a hand-picked fallback for when you don't have a
// signature set loaded yet to compute DeriveWeights from. Table 9's own
// "Forced Negotiation" and "Multi-Packet Initial" rows show every device
// bypassing both -- i.e. they behave as reachability canaries rather than
// discriminators for these five devices -- so this downweights just those
// two as a starting intuition; DeriveWeights(sigs) supersedes this once you
// have real data; both are trivially overridable by passing your own
// Weights value to MatchSignature/RankSignatures instead.
func DefaultWeights() Weights {
	return Weights{
		"quic_feature_forced_negotiation":   0.25,
		"quic_feature_multi_packet_initial": 0.25,
	}
}

// MatchResult is one Signature's comparison against an observed Outcome map.
type MatchResult struct {
	Device        string  `json:"device"`
	Score         float64 `json:"score"`      // weighted agreement fraction, 0..1
	Agreements    int     `json:"agreements"` // vectors where observed == reference
	Disagreements int     `json:"disagreements"`
	Compared      int     `json:"compared"` // vectors where BOTH sides had a known Outcome
	Confidence    string  `json:"confidence"`
}

// confidenceFor turns a comparison count into a plain-language caveat. These
// thresholds are deliberately conservative and easy to change; they exist so
// a 3-vector "match" doesn't read as authoritative as a 25-vector one.
func confidenceFor(compared, totalPossible int) string {
	switch {
	case compared < totalPossible/3:
		return "low (few overlapping vectors; treat as a hint, not an identification)"
	case compared < 2*totalPossible/3:
		return "medium"
	default:
		return "high (most vectors had reference data to compare against)"
	}
}

// MatchSignature scores one Signature against observed, an Outcome map as
// produced by OutcomeMap. Vectors where either side is OutcomeUnknown or
// OutcomeInconsistent are skipped entirely -- they neither help nor hurt
// the score -- so Compared tells you how much the resulting Score is
// actually worth.
func MatchSignature(observed map[string]Outcome, sig Signature, weights Weights, totalPossible int) MatchResult {
	var agreeW, totalW float64
	var agreements, disagreements int
	for name, ref := range sig.Vectors {
		if ref == OutcomeUnknown || ref == OutcomeInconsistent {
			continue
		}
		obs, ok := observed[name]
		if !ok || obs == OutcomeUnknown || obs == OutcomeInconsistent {
			continue
		}
		w := weights.get(name)
		totalW += w
		if obs == ref {
			agreements++
			agreeW += w
		} else {
			disagreements++
		}
	}
	compared := agreements + disagreements
	score := 0.0
	if totalW > 0 {
		score = agreeW / totalW
	}
	return MatchResult{
		Device:        sig.Device,
		Score:         score,
		Agreements:    agreements,
		Disagreements: disagreements,
		Compared:      compared,
		Confidence:    confidenceFor(compared, totalPossible),
	}
}

// RankSignatures scores every signature in sigs against observed and
// returns them sorted best-match-first. Ties are broken by Compared (more
// overlapping evidence wins) since Score alone is easy to game with very
// few compared vectors.
func RankSignatures(observed map[string]Outcome, sigs []Signature, weights Weights) []MatchResult {
	totalPossible := len(observed)
	results := make([]MatchResult, 0, len(sigs))
	for _, s := range sigs {
		if isMetaSignature(s) {
			continue
		}
		results = append(results, MatchSignature(observed, s, weights, totalPossible))
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Compared > results[j].Compared
	})
	return results
}
