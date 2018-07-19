FROM ubuntu:16.04 AS build

RUN apt-get update && \
    apt-get install -y software-properties-common git

##
## GO
##
RUN apt-get update && \
    add-apt-repository ppa:gophers/archive && \
    apt-get update && \
    apt-get install -y golang-1.10-go
RUN ln -s /usr/lib/go-1.10/bin/go /usr/bin/go

ENV GOROOT /usr/lib/go-1.10
ENV GOPATH /root/go
ENV PATH $PATH:${GOROOT}/bin:${GOPATH}/bin

##
## govendor
##
RUN go get -u github.com/kardianos/govendor

##
## build
##
WORKDIR ${GOPATH}/src/github.com/yarencheng/crypto-trade/go
RUN pwd
COPY ./go .
RUN go install -race -v ./...
RUN cp ${GOPATH}/bin/poloniex_recorder /poloniex_recorder

FROM ubuntu:16.04 AS runtime

RUN apt-get update && \
    apt-get install -y ca-certificates

COPY --from=build /poloniex_recorder /bin/.
ENTRYPOINT [ "poloniex_recorder" ]