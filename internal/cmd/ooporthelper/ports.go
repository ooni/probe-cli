package main

//
// List of ports we want to use for running integration tests
//

// Ports for testing the testhelper
// Note: we must only use unprivileged ports here to ensure tests run successfully
var TestPorts = []string{
	"8080", // tcp
	"5050", // tcp
}
