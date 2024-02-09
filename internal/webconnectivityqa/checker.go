package webconnectivityqa

import (
	"errors"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/x/dslx"
)

// Checker checks whether a measurement is correct.
type Checker interface {
	Check(mx *model.Measurement) error
}

// ReadWriteEventsExistentialChecker fails if there are zero network events.
type ReadWriteEventsExistentialChecker struct{}

var _ Checker = &ReadWriteEventsExistentialChecker{}

// ErrCheckerNoReadWriteEvents indicates that a checker did not find any read/write events.
var ErrCheckerNoReadWriteEvents = errors.New("no read or write events")

// ErrCheckerUnexpectedWebConnectivityVersion indicates that the version is unexpected
var ErrCheckerUnexpectedWebConnectivityVersion = errors.New("unexpected Web Connectivity version")

// Check implements Checker.
func (*ReadWriteEventsExistentialChecker) Check(mx *model.Measurement) error {
	// we don't care about v0.4
	if strings.HasPrefix(mx.TestVersion, "0.4.") {
		return nil
	}

	// make sure it's v0.5
	if !strings.HasPrefix(mx.TestVersion, "0.5.") {
		return ErrCheckerUnexpectedWebConnectivityVersion
	}

	// serialize and reparse the test keys
	var tk *dslx.Observations
	must.UnmarshalJSON(must.MarshalJSON(mx.TestKeys), &tk)

	// count the read/write events
	var count int
	for _, ev := range tk.NetworkEvents {
		switch ev.Operation {
		case netxlite.ReadOperation, netxlite.WriteOperation:
			count++
		default:
			// nothing
		}
	}

	// make sure there's at least one network event
	if count <= 0 {
		return ErrCheckerNoReadWriteEvents
	}
	return nil
}
