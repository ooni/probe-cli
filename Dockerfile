FROM openobservatory/mk-alpine:latest
RUN apk add --no-progress git go
ADD . /oonibuild
