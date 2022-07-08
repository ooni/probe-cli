package measurexlite

//
// Logging support
//

import "github.com/ooni/probe-cli/v3/internal/measurex"

// TODO(bassosimone): we should eventually remove measurex and
// move the logging code from measurex to this package.

// NewOperationLogger is an alias for measurex.NewOperationLogger.
var NewOperationLogger = measurex.NewOperationLogger

// OperationLogger is an alias for measurex.OperationLogger.
type OperationLogger = measurex.OperationLogger
