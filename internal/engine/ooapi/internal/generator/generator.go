// Command generator generates code in the ooapi package.
//
// To this end, it uses the content of the apimodel package as
// well as the content of the spec.go file.
//
// The apimodel package defines the model, i.e., the structure
// of requests and responses and how messages should be sent
// and received.
//
// The spec.go file describes all the implemented APIs.
//
// If you change apimodel or spec.go, remember to run the
// `go generate ./...` command to regenerate all files.
package main

import (
	"flag"
	"fmt"
)

var flagFile = flag.String("file", "", "Indicate which file to regenerate")

func main() {
	flag.Parse()
	switch file := *flagFile; file {
	case "apis.go":
		GenAPIsGo(file)
	case "responses.go":
		GenResponsesGo(file)
	case "requests.go":
		GenRequestsGo(file)
	case "swagger_test.go":
		GenSwaggerTestGo(file)
	case "apis_test.go":
		GenAPIsTestGo(file)
	case "callers.go":
		GenCallersGo(file)
	case "caching.go":
		GenCachingGo(file)
	case "login.go":
		GenLoginGo(file)
	case "cloners.go":
		GenClonersGo(file)
	case "fakeapi_test.go":
		GenFakeAPITestGo(file)
	case "caching_test.go":
		GenCachingTestGo(file)
	case "login_test.go":
		GenLoginTestGo(file)
	case "clientcall.go":
		GenClientCallGo(file)
	default:
		panic(fmt.Sprintf("don't know how to create this file: %s", file))
	}
}
