package main

import "os"

func main() {
	antani := os.Getenv("ANTANI")
	mascetti := os.Getenv("MASCETTI")
	stuzzica := os.Getenv("STUZZICA")
	if antani != "antani" {
		os.Exit(1)
	}
	if mascetti != "mascetti" {
		os.Exit(2)
	}
	if stuzzica != "stuzzica" {
		os.Exit(3)
	}
}
