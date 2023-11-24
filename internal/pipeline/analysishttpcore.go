package pipeline

import "github.com/ooni/probe-cli/v3/internal/optional"

func (ax *Analysis) httpExperimentFailureHelper(db *DB, fx func(txId int64, probeFailure Failure)) {
	// skip if there's no final request
	if db.WebFinalRequest.IsNone() {
		return
	}
	probeFR := db.WebFinalRequest.Unwrap()

	// skip if the HTTP failure is not defined (bug?)
	if probeFR.HTTPFailure.IsNone() {
		return
	}

	// skip if the final request succeded
	// TODO(bassosimone): say that the probe succeeds and the TH fails, then what?
	probeFailure := probeFR.HTTPFailure.Unwrap()
	if probeFailure == "" {
		return
	}

	// skip if the final request is not defined for the TH
	if db.THWeb.IsNone() {
		return
	}
	thFR := db.THWeb.Unwrap()

	// skip if the failure is not defined for the TH
	if thFR.HTTPFailure.IsNone() {
		return
	}

	// skip if also the TH's HTTP request failed
	thFailure := thFR.HTTPFailure.Unwrap()
	if thFailure != "" {
		return
	}

	// invoke user defined func
	fx(probeFR.TransactionID, probeFailure)
}

// ComputeHTTPUnexpectedFailure computes HTTPUnexpectedFailure.
func (ax *Analysis) ComputeHTTPUnexpectedFailure(db *DB) {
	ax.httpExperimentFailureHelper(db, func(txId int64, probeFailure Failure) {
		ax.HTTPUnexpectedFailure = append(ax.HTTPUnexpectedFailure, txId)
	})
}

// ComputeHTTPExperimentFailure computes HTTPExperimentFailure.
func (ax *Analysis) ComputeHTTPExperimentFailure(db *DB) {
	ax.httpExperimentFailureHelper(db, func(txId int64, probeFailure Failure) {
		ax.HTTPExperimentFailure = optional.Some(probeFailure)
	})
}
