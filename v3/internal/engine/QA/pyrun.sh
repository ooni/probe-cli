#!/bin/sh
set -ex
export GOPATH=/jafar/QA/GOPATH GOCACHE=/jafar/QA/GOCACHE GO111MODULE=on
go build -v ./cmd/miniooni
go build -v ./cmd/jafar
sudo ./QA/$1.py ./miniooni
