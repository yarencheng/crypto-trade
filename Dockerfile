FROM golang:1.10.3-stretch AS GO
RUN go get -u github.com/kardianos/govendor

WORKDIR /go/src/github.com/yarencheng/crypto-trade/go
COPY ./go .
RUN ls -ltr
RUN govendor sync -v

RUN go test -v -race ./...
RUN go install ./cmd/simple_trader/...

FROM alpine:3.8
RUN which ls
