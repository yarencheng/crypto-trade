FROM golang:1.10.3-stretch AS GO
WORKDIR /go/src/github.com/yarencheng/crypto-trade/

COPY ./go go/

WORKDIR /go/src/github.com/yarencheng/crypto-trade/go

RUN go get -u github.com/kardianos/govendor
RUN govendor sync -v


RUN go test ./...

RUN go install ./cmd/simple_trader/...

FROM alpine:3.8
RUN which ls
