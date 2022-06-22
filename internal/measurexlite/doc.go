// Package measurexlite contains measurement extensions.
//
// See docs/design/dd-003-step-by-step.md in the ooni/probe-cli
// repository for the design document.
//
// This implementation features a Trace that saves events in
// buffered channels as proposed by df-003-step-by-step.md. We
// have reasonable default buffers for channels. But, if you
// are not draining them, eventually we stop collecting events.
package measurexlite
