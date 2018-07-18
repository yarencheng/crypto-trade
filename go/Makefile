## command
GO           = go
GO_VENDOR    = govendor

################################################

.PHONY: all
all: build test

.PHONY: check
check: check-govendor

.PHONY: pre-build
pre-build: check
	$(GO_VENDOR) sync -v
	$(GO_VENDOR) remove -v +unused

.PHONY: build
build: pre-build
	$(GO) build -race -v ./...

.PHONY: test
test: build
	$(GO) test -race -v -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html

.PHONY: install
install: build
	$(GO) install -race -v ./...

.PHONY: clean
clean:
	$(GO) clean -i -r -x -cache -testcache
	$(RM) -f coverage.txt coverage.html

## check build env #############################

.PHONY: check-govendor
check-govendor:
	$(info check govendor)
	@[ "`which $(GO_VENDOR)`" != "" ] || (echo "govendor is missing"; false) && (echo "   .. OK")

