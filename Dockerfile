FROM golang:1.10.3-alpine3.7 AS GO
RUN apk add --no-cache git
RUN go get -u github.com/kardianos/govendor

WORKDIR /go/src/github.com/yarencheng/crypto-trade/go
COPY ./go .
RUN govendor sync -v
RUN go test -v ./...
RUN go install ./cmd/simple_trader/...

FROM alpine:3.8
COPY --from=GO /go/bin/simple_trader /bin/.

ENTRYPOINT [ "simple_trader" ]
