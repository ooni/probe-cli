# Package github.com/ooni/probe-engine/probeservices

This package contains code to contact OONI probe services.

The probe services are HTTPS endpoints distributed across a bunch of data
centres implementing a bunch of OONI APIs. When started, OONI will benchmark
the available probe services and select the fastest one. Eventually all the
possible OONI APIs will run as probe services.
