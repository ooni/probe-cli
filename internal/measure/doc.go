// Package measure contains measuring primitives.
//
// TL;DR: we separate DNS resolution from accessing endpoints
// because we want to measure all the endpoints for a given DNS
// name (at least for websteps). This library defines all the
// fundamental and compounded operations required to fullfil this
// goal of measuring DNS resolution and endpoints. The entry
// point for using this library is the Measurer struct.
//
// To understand how to use this library, we recommend reading
// the ./internal/tutorial/measure tutorial that explains the
// general design and provides usage examples.
package measure

/*
	TODO:

	1. [x] tracing

	2. [ ] data format

	3. [x] parroting

	4. [x] dns transport

	5. [x] flows

	6. [x] HTTP code is ugly
*/
