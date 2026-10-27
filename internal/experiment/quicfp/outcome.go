package quicfp

import "encoding/json"

// Outcome is a per-vector classification: did the probed vector reach the
// target and get answered (Bypassed), or not (Blocked)? This is distinct
// from SinglePing.Blocked, which is a per-repetition raw signal; Outcome is
// the aggregate across repetitions that diff.go and fingerprint.go operate
// on.
type Outcome int

const (
	// OutcomeUnknown means we have no data (e.g. zero repetitions ran, or
	// every repetition errored before send).
	OutcomeUnknown Outcome = iota
	// OutcomeBypassed means a response was observed for most repetitions:
	// the vector reached the target and the target reacted as a
	// non-interfered path would.
	OutcomeBypassed
	// OutcomeBlocked means no response was observed for most
	// repetitions: something between the probe and the target consumed
	// the vector without a trace.
	OutcomeBlocked
	// OutcomeInconsistent means repetitions disagreed with each other
	// closely enough that neither Bypassed nor Blocked cleared the
	// classification threshold (see AggregateOutcome). This usually
	// means the deadline/repetition count needs tuning, or that
	// something is *rate-limiting* rather than deterministically
	// blocking.
	OutcomeInconsistent
)

func (o Outcome) String() string {
	switch o {
	case OutcomeBypassed:
		return "bypassed"
	case OutcomeBlocked:
		return "blocked"
	case OutcomeInconsistent:
		return "inconsistent"
	default:
		return "unknown"
	}
}

// MarshalJSON renders Outcome as its lowercase name, so signature files and
// diff reports stay human-editable.
func (o Outcome) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

// UnmarshalJSON parses the lowercase names back into an Outcome, so hand-
// written or paper-derived signature files are easy to author.
func (o *Outcome) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "bypassed":
		*o = OutcomeBypassed
	case "blocked":
		*o = OutcomeBlocked
	case "inconsistent":
		*o = OutcomeInconsistent
	default:
		*o = OutcomeUnknown
	}
	return nil
}

// DefaultAgreementThreshold is the fraction of repetitions that must agree
// before AggregateOutcome commits to Bypassed or Blocked, rather than
// reporting Inconsistent. 2/3 matches the paper's own measurement
// methodology (Section 6.4): "classified a length permutation as censored
// only if [the] behavior was observed in at least two-thirds of trials."
const DefaultAgreementThreshold = 2.0 / 3.0

// AggregateOutcome reduces a vector's repeated SinglePing results to one
// Outcome, requiring at least threshold agreement (see
// DefaultAgreementThreshold) before committing to Bypassed or Blocked.
func AggregateOutcome(pings []*SinglePing, threshold float64) Outcome {
	// Only pings that actually got a chance to reach the network count;
	// a build/send error before anything left the socket tells us
	// nothing about the path and must not be conflated with "blocked".
	var attempted, blocked int
	for _, p := range pings {
		if p.Failure != nil && len(p.Responses) == 0 && !p.Blocked {
			continue // build/send error, not a network signal
		}
		attempted++
		if p.Blocked {
			blocked++
		}
	}
	if attempted == 0 {
		return OutcomeUnknown
	}
	frac := float64(blocked) / float64(attempted)
	switch {
	case frac >= threshold:
		return OutcomeBlocked
	case frac <= 1-threshold:
		return OutcomeBypassed
	default:
		return OutcomeInconsistent
	}
}

// AggregateVectorResult reduces one VectorResult's Pings to a single
// Outcome using DefaultAgreementThreshold.
func AggregateVectorResult(vr *VectorResult) Outcome {
	return AggregateOutcome(vr.Pings, DefaultAgreementThreshold)
}

// OutcomeMap builds a vector-name -> Outcome map from a TestKeys result,
// the shape both diff.go and fingerprint.go consume.
func OutcomeMap(tk *TestKeys) map[string]Outcome {
	out := make(map[string]Outcome, len(tk.Vectors))
	for _, vr := range tk.Vectors {
		out[vr.Name] = AggregateVectorResult(vr)
	}
	return out
}
