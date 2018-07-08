# FROM golang:1.10.3-alpine3.7 AS GO
# RUN apk add --no-cache git gcc libc-dev
# RUN go get -u github.com/kardianos/govendor

# WORKDIR /go/src/github.com/yarencheng/crypto-trade/go
# COPY ./go .
# RUN govendor sync -v
# RUN go test -timeout 10s -v ./...
# RUN go install ./cmd/bitfinex_record/...

# FROM alpine:3.8
# COPY --from=GO /go/bin/bitfinex_record /bin/.

# ENTRYPOINT [ "bitfinex_record" ]


FROM golang:1.10.3-alpine AS GO
RUN apk add --no-cache git gcc libc-dev make
RUN go get -u github.com/kardianos/govendor

WORKDIR /go/src/github.com/yarencheng/crypto-trade/go
COPY ./go .
RUN make install

FROM alpine:3.8
RUN apk add --no-cache ca-certificates
COPY --from=GO /go/bin/bitfinex_record /bin/.

ENTRYPOINT [ "bitfinex_record" ]
