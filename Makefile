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

## docker ############

.PHONY: docker.bitfinex_record.build
docker.bitfinex_record.build:
	docker build -t yarencheng/crypto-trade/bitfinex_record:latest --file docker/bitfinex_record/Dockerfile .

## check build env #############################

.PHONY: check-docker
check-docker:
	$(info check $(DOCKER))
	@[ "`which $(DOCKER)`" != "" ] || (echo "docker is missing"; false) && (echo "   .. OK")


