package main

import (
	"fmt"
	"testing"
)

func init() {
	*reportid = `20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU`
	*input = `https://www.example.org`
}

func TestRaw(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	*mode = "raw"
	main()
}

func TestMeta(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	*mode = "meta"
	main()
}

func TestInvalidMode(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("the code did not panic")
		}
	}()
	osExit = func(code int) {
		panic(fmt.Errorf("%d", code))
	}
	*mode = "antani"
	main()
}
