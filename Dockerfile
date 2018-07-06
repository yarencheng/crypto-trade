FROM golang:1.10.3-stretch AS GO
RUN go get -u github.com/kardianos/govendor

WORKDIR /go/src/github.com/yarencheng/crypto-trade/
COPY ./ .
RUN ls -ltr
RUN govendor sync -v

RUN govendor list
RUN govendor remove +unused
RUN govendor list

RUN go test ./...
RUN go install ./go/cmd/simple_trader/...

FROM alpine:3.8
RUN which ls
