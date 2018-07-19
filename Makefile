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
pre-build: check
	$(MAKE) pre-build -C go

.PHONY: build
build: pre-build docker.bitfinex_recorder.build docker.poloniex_recorder.build
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
	$(DOCKER) rmi yarencheng/crypto-trade/bitfinex_recorder:latest || true
	$(DOCKER) rmi yarencheng/crypto-trade/poloniex_recorder:latest || true

## docker ############

.PHONY: docker.bitfinex_recorder.build
docker.bitfinex_recorder.build:
	docker build -t yarencheng/crypto-trade/bitfinex_recorder:latest --file docker/bitfinex_recorder/Dockerfile .
	docker build -t yarencheng/crypto-trade/bitfinex_recorder:latest-debug --file docker/bitfinex_recorder/debug.Dockerfile .

.PHONY: docker.poloniex_recorder.build
docker.poloniex_recorder.build:
	docker build -t yarencheng/crypto-trade/poloniex_recorder:latest --file docker/poloniex_recorder/Dockerfile .
	docker build -t yarencheng/crypto-trade/poloniex_recorder:latest-debug --file docker/poloniex_recorder/debug.Dockerfile .

## check build env #############################

.PHONY: check-docker
check-docker:
	$(info check $(DOCKER))
	@[ "`which $(DOCKER)`" != "" ] || (echo "docker is missing"; false) && (echo "   .. OK")


