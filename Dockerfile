FROM golang:1.14-alpine
RUN apk add --no-progress gcc git linux-headers musl-dev
