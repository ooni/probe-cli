# Code to build C dependencies

This directory contains code to build C dependencies.

We have a directory for each dependency. The directory contains zero or
more patches, and a generic bash build script called `build`.

The [build-android](build-android) script sets the proper configuration
variables to build packages for android and calls individual `build` scripts.

A future version of this directory will contain additional system
specific build scripts such as `build-ios`, which will behave like
`build-android` but would obviously target another system.
