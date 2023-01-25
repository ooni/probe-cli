package main

// cdepsEnv contains the environment for compiling a C dependency.
type cdepsEnv struct {
	// cflags contains the CFLAGS to use when compiling.
	cflags []string

	// destdir is the directory where to install.
	destdir string

	// openSSLCompiler is the compiler name for OpenSSL.
	openSSLCompiler string
}
