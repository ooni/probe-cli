package main

import "testing"

func TestSmoke(t *testing.T) {
	*target = "http://www.example.com"
	main()
}
