# Package github.com/ooni/probe-engine/oonimkall

Package oonimkall implements APIs used by OONI mobile apps. We
expose these APIs to mobile apps using gomobile.

We expose two APIs: the task API, which is derived from the
API originally exposed by Measurement Kit, and the session API,
which is a Go API that mobile apps can use via `gomobile`.

This package is named oonimkall because it contains a partial
reimplementation of the mkall API implemented by Measurement Kit
in, e.g., [mkall-ios](https://github.com/measurement-kit/mkall-ios).

The basic tenet of the task API is that you define an experiment
task you wanna run using a JSON, then you start a task for it, and
you receive events as serialized JSONs. In addition to this
functionality, we also include extra APIs used by OONI mobile.

The basic tenet of the session API is that you create an instance
of `Session` and use it to perform the operations you need.
