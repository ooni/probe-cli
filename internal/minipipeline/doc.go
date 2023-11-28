// Package minipipeline implements a minimal data processing pipeline used
// to analyze local measurements collected by OONI Probe.
//
// This package mimics ooni/data design.
//
// A user provides as input to the minipipeline an OONI measurement and obtains
// as an intermediate result one or more observations. In turn, the user
// can process the observations to obtain a measurement analysis. Observations
// are an intermediate, flat data format useful to simplify writing analysis
// algorithms. The measurement analysis contains scalar, vector, and map
// fields summarizing the measurement. Each experiment should write custom
// expressions for generating top-level test keys given the analysis.
//
// For [*WebMeasurement], [*WebObservation] is an observation. The
// [*WebObservationsContainer] type allows one to create observations
// from OONI experiments measurements. In the same vein, the [*WebAnalysis]
// type contains the analysis for [*WebMeasurement].
//
// The [IngestWebMeasurement] convenience function simplifies transforming
// a [*WebMeasurement] into a [*WebObservationsContainer]. Likewise, the
// [AnalyzeWebObservations] function simplifies obtaining a [*WebAnalysis].
package minipipeline
