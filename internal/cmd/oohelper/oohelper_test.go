package main

import "testing"

func TestSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	*target = "http://www.example.com"
	main()
}
