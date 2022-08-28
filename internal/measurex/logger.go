package measurex

import "github.com/ooni/probe-cli/v3/internal/measurexlite"

//
// Logger
//
// Code for logging
//

// NewOperationLogger is an alias for measurex.NewOperationLogger.
var NewOperationLogger = measurexlite.NewOperationLogger

// OperationLogger is an alias for measurex.OperationLogger.
type OperationLogger = measurexlite.OperationLogger
