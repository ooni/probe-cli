// Code generated by go generate; DO NOT EDIT.
// 2021-06-15 10:55:57.987741 +0200 CEST m=+0.000211126

package ooapi

//go:generate go run ./internal/generator -file cloners.go

// clonerForPsiphonConfigAPI represents any type exposing a method
// like simplePsiphonConfigAPI.WithToken.
type clonerForPsiphonConfigAPI interface {
	WithToken(token string) callerForPsiphonConfigAPI
}

// clonerForTorTargetsAPI represents any type exposing a method
// like simpleTorTargetsAPI.WithToken.
type clonerForTorTargetsAPI interface {
	WithToken(token string) callerForTorTargetsAPI
}
