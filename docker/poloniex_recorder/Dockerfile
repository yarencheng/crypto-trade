FROM golang:1.10.3-alpine AS GO
RUN apk add --no-cache git gcc libc-dev make
RUN go get -u github.com/kardianos/govendor

WORKDIR /go/src/github.com/yarencheng/crypto-trade/go
COPY ./go .
RUN make install

FROM alpine:3.8
RUN apk add --no-cache ca-certificates
COPY --from=GO /go/bin/bitfinex_recorder /bin/.

ENTRYPOINT [ "bitfinex_recorder" ]
