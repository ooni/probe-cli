// Package oonimkall implements APIs used by OONI mobile apps. We
// expose these APIs to mobile apps using gomobile.
//
// We expose two APIs: the task API, which is derived from the
// API originally exposed by Measurement Kit, and the session API,
// which is a Go API that mobile apps can use via `gomobile`.
//
// This package is named oonimkall because it contains a partial
// reimplementation of the mkall API implemented by Measurement Kit
// in, e.g., https://github.com/measurement-kit/mkall-ios.
//
// Semantic versioning policy
//
// This package is public for technical reasons. We cannot use `go
// mobile` on a private package. Yet, we are not going to bump this
// repository's major number in case we change oonimkall's API. We
// consider this package our private API for interfacing with our
// mobile applications for Android and iOS.
//
// Task API
//
// The basic tenet of the task API is that you define an experiment
// task you wanna run using a JSON, then you start a task for it, and
// you receive events as serialized JSONs. In addition to this
// functionality, we also include extra APIs used by OONI mobile.
//
// The task API was first defined in Measurement Kit v0.9.0. In this
// context, it was called "the FFI API". The API we expose here is not
// strictly an FFI API, but is close enough for the purpose of using
// OONI from Android and iOS. See https://git.io/Jv4Rv
// (measurement-kit/measurement-kit@v0.10.9) for a comprehensive
// description of MK's FFI API.
//
// See also https://github.com/ooni/probe-engine/pull/347 for the
// design document describing the task API.
//
// See also https://github.com/ooni/probe-cli/v3/internal/engine/blob/master/DESIGN.md,
// which explains why we implemented the oonimkall API.
//
// Session API
//
// The Session API is a Go API that can be exported to mobile apps
// using the gomobile tool. The latest design document for this API is
// at https://github.com/ooni/probe-engine/pull/954.
//
// The basic tenet of the session API is that you create an instance
// of `Session` and use it to perform the operations you need.
package oonimkall
