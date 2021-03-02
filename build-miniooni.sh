#!/bin/sh
set -e
case $1 in
  macos|darwin)
    go run ./internal/cmd/getresources
    export GOOS=darwin GOARCH=amd64
    go build -o ./CLI/darwin/amd64 -ldflags="-s -w" ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/darwin/amd64/miniooni";;
  linux)
    go run ./internal/cmd/getresources
    export GOOS=linux GOARCH=386
    go build -o ./CLI/linux/386 -tags netgo -ldflags='-s -w -extldflags "-static"' ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/linux/386/miniooni"
    export GOOS=linux GOARCH=amd64
    go build -o ./CLI/linux/amd64 -tags netgo -ldflags='-s -w -extldflags "-static"' ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/linux/amd64/miniooni"
    export GOOS=linux GOARCH=arm GOARM=7
    go build -o ./CLI/linux/arm -tags netgo -ldflags='-s -w -extldflags "-static"' ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/linux/arm/miniooni"
    export GOOS=linux GOARCH=arm64
    go build -o ./CLI/linux/arm64 -tags netgo -ldflags='-s -w -extldflags "-static"' ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/linux/arm64/miniooni";;
  windows)
    go run ./internal/cmd/getresources
    export GOOS=windows GOARCH=386
    go build -o ./CLI/windows/386 -ldflags="-s -w" ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/windows/386/miniooni.exe"
    export GOOS=windows GOARCH=amd64
    go build -o ./CLI/windows/amd64 -ldflags="-s -w" ./internal/cmd/miniooni
    echo "Binary ready at ./CLI/windows/amd64/miniooni.exe";;
  *)
    echo "usage: $0 darwin|linux|windows" 1>&2
    exit 1
esac
