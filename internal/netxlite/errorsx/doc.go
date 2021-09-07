// Package errorsx contains code to classify errors.
//
// We define the ErrWrapper type, that should wrap any error
// and map it to the corresponding OONI failure.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md
// for a list of OONI failure strings.
//
// We define ClassifyXXX functions that map an `error` type to
// the corresponding OONI failure.
//
// When we cannot map an error to an OONI failure we return
// an "unknown_failure: XXX" string where the XXX part has
// been scrubbed so to remove any network endpoints.
//
// The general approach we have been following for this
// package has been to return the same strings that we used
// with the previous measurement engine, Measurement Kit
// available at https://github.com/measurement-kit/measurement-kit.
package errorsx
