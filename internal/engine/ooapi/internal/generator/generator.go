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

func main() {
	// TODO(bassosimone): the current generator algorithm is not
	// efficient because we re-generate all files each time. We
	// should instead use a flag to tell the code which is the file
	// we want to regenerate for each invocation.
	GenAPIsGo()
	GenResponsesGo()
	GenRequestsGo()
	GenSwaggerGo()
	GenAPIsTestGo()
	GenCallersGo()
	GenCachingGo()
	GenLoginGo()
	GenClonersGo()
	GenFakeAPITestGo()
	GenCachingTestGo()
	GenLoginTestGo()
}
