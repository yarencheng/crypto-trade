## command
GO           = go
GO_VENDOR    = govendor
DOCKER       = docker

###########################################

.PHONY: all
all: build test

.PHONY: check
check: check-docker
	$(MAKE) check -C go

.PHONY: pre-build
pre-build: check antlr4
	$(MAKE) pre-build -C go

.PHONY: build
build: pre-build docker.bitfinex_record.build
	$(MAKE) build -C go

.PHONY: test
test: build
	$(MAKE) test -C go

.PHONY: install
install: build
	$(MAKE) install -C go

.PHONY: clean
clean:
	$(MAKE) clean -C go
	$(DOCKER) rmi yarencheng/crypto-trade/bitfinex_record:latest || true
	$(DOCKER) rmi yarencheng/crypto-trade/antlr4:latest || true

## docker ############

.PHONY: docker.bitfinex_record.build
docker.bitfinex_record.build:
	docker build -t yarencheng/crypto-trade/bitfinex_record:latest --file docker/bitfinex_record/Dockerfile .

.PHONY: docker.antlr4.build
docker.antlr4.build:
	docker build -t yarencheng/crypto-trade/antlr4:latest --file docker/antlr4/Dockerfile .

## antlr4 ############

.PHONY: antlr4
antlr4: docker.antlr4.build
	docker run -it --rm \
		-w `pwd` \
		--volume `pwd`:`pwd` \
		--user `id -u`:`id -g` \
		yarencheng/crypto-trade/antlr4:latest \
			-Dlanguage=Go \
			-o go/exchange/poloniex/parser/ \
			-long-messages \
			-package antlr4 \
			antlr4/JSON.g4

## check build env #############################

.PHONY: check-docker
check-docker:
	$(info check $(DOCKER))
	@[ "`which $(DOCKER)`" != "" ] || (echo "docker is missing"; false) && (echo "   .. OK")


