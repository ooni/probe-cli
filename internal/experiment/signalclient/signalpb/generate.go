// Package signalpb contains the protobuf messages of Signal's WebSocket protocol (signal.proto).
package signalpb

// Needs protoc and protoc-gen-go (the google.golang.org/protobuf version in go.mod) on the PATH.
//go:generate protoc --go_out=. --go_opt=paths=source_relative signal.proto
