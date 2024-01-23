package torcontrol

//
// status.go - enumerates tor control status codes.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

// The various control port StatusCode constants.
const (
	StatusOk            = 250
	StatusOkUnnecessary = 251

	StatusErrResourceExhausted      = 451
	StatusErrSyntaxError            = 500
	StatusErrUnrecognizedCmd        = 510
	StatusErrUnimplementedCmd       = 511
	StatusErrSyntaxErrorArg         = 512
	StatusErrUnrecognizedCmdArg     = 513
	StatusErrAuthenticationRequired = 514
	StatusErrBadAuthentication      = 515
	StatusErrUnspecifiedTorError    = 550
	StatusErrInternalError          = 551
	StatusErrUnrecognizedEntity     = 552
	StatusErrInvalidConfigValue     = 553
	StatusErrInvalidDescriptor      = 554
	StatusErrUnmanagedEntity        = 555

	StatusAsyncEvent = 650
)
