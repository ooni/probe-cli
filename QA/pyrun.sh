#!/bin/sh
set -ex
export GOPATH=/jafar/QA/GOPATH GOCACHE=/jafar/QA/GOCACHE GO111MODULE=on
go run ./internal/cmd/getresources
go build -v ./internal/cmd/miniooni
go build -v ./internal/cmd/jafar
sudo ./QA/$1.py ./miniooni
