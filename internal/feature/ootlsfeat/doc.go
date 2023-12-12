// Package oohttpfeat implements the ootls feature. When it is possible
// to enable this feature, we include mitigations that make TLS more robust
// for Android, as documented by the following blog post
// https://ooni.org/post/making-ooni-probe-android-more-resilient/. Otherwise,
// we will just use the standard library.
package ootlsfeat
