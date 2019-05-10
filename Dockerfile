FROM openobservatory/mk-alpine:20190509
RUN apk add --no-progress git go
ADD . /oonibuild
