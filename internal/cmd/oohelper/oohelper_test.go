package main

import "testing"

func TestSmoke(t *testing.T) {
	*debug = true // To help with https://github.com/ooni/probe/issues/1409
	*target = "http://www.example.com"
	main()
}
