// Package model contains the shared interfaces and data structures.
//
// # Criteria for adding a type to this package
//
// This package should contain two types:
//
// 1. important interfaces that are shared by several packages
// within the codebase, with the objective of separating unrelated
// pieces of code and making unit testing easier;
//
// 2. important pieces of data that are shared across different
// packages (e.g., the representation of a Measurement).
//
// In general, this package should not contain logic, unless
// this logic is strictly related to data structures and we
// cannot implement this logic elsewhere.
//
// # Content of this package
//
// The following list (which may not always be up-to-date)
// summarizes the categories of types that currently belong here
// and names the files in which they are implemented:
//
// - experiment.go: generic definition of a network experiment
// and all the required support types;
//
// - keyvaluestore.go: generic definition of a key-value store,
// used in several places across the codebase;
//
// - logger.go: generic definition of an apex/log compatible logger,
// used in several places across the codebase;
//
// - measurement.go: data type representing the result of
// a network measurement, used in many many places;
//
// - netx.go: network extension interfaces and data used everywhere
// we need to perform network operations;
//
// - ooapi.go: types to communicate with the OONI API.
package model
