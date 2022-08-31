package main

//
// List of ports we want to test for filtering
//

// Note: Using privileged ports like :80 requires elevated permissions
var Ports = []string{
	":22",  // tcp
	":23",  // tcp
	":25",  // tcp
	":80",  // tcp
	":143", // tcp
	":443", // tcp
	":445", // tcp
	":587", // tcp
	":993", // tcp
}

// Ports for testing the testhelper
// Note: we must only use unprivileged ports here to ensure tests run successfully
var TestPorts = []string{
	":8080", //tcp
	":5050", //tcp
}
