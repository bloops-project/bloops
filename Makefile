DOCKER ?= docker
PROTOC ?= protoc
GO_CMD?=go
CI_LINT ?= golangci-lint

CGO_ENABLED?=0
GOOS?=linux
GOARCH=amd64
GO111MODULE?=on

BUILD_INFO_PACKAGE = github.com/bloops-games/bloops/internal/buildinfo
BUILD_TAG=$(shell git describe --tags --abbrev=0)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d-%H:%M')
BUILD_NAME_CLI?=bloopsbot-cli
BUILD_NAME_SRV?=bloopsbot-srv

unittest:
	@$(GO_CMD) test -short $$(go list ./... | grep -v /vendor/)

test:
	@$(GO_CMD) test -v -cover -covermode=atomic ./...

test-cover:
	@$(GO_CMD) test -count=2 -race -timeout=10m ./... -coverprofile=coverage.out

.PHONY: srv
srv:
	GOARCH=${GOARCH} GO111MODULE=${GO111MODULE} CGO_ENABLED=0 GOOS=${GOOS} \
$(GO_CMD) build -o build/bloops-srv -trimpath \
-ldflags "-s -w -X ${BUILD_INFO_PACKAGE}.BuildTag=${BUILD_TAG} -X ${BUILD_INFO_PACKAGE}.Time=${BUILD_TIME} -X ${BUILD_INFO_PACKAGE}.Name=${BUILD_NAME_CLI}" \
./cmd/bloops-srv

.PHONY: cli
cli:
	GOARCH=${GOARCH} GO111MODULE=${GO111MODULE} CGO_ENABLED=0 GOOS=${GOOS} \
$(GO_CMD) build -o build/bloops-cli -trimpath \
-ldflags "-s -w -X ${BUILD_INFO_PACKAGE}.BuildTag=${BUILD_TAG} -X ${BUILD_INFO_PACKAGE}.Time=${BUILD_TIME} -X ${BUILD_INFO_PACKAGE}.Name=${BUILD_NAME_CLI}" \
./cmd/bloops-cli

docker:
	@$(DOCKER) build -t bloops .

vet:
	@$(GO_CMD) list -f '{{.Dir}}' ./... | grep -v /vendor/ \
		| grep -v '.*github.com/bloops-games/bloops$$' \
		| xargs $(GO_CMD) vet ; if [ $$? -eq 1 ]; then \
			echo ""; \
			echo "Vet found suspicious constructs. Please check the reported constructs"; \
			echo "and fix them if necessary before submitting the code for reviewal."; \
		fi

lint:
	@$(GO_CMD) list -f '{{.Dir}}' ./... | grep -v /vendor/ \
		| xargs golangci-lint run; if [ $$? -eq 1 ]; then \
			echo ""; \
			echo "Lint found suspicious constructs. Please check the reported constructs"; \
			echo "and fix them if necessary before submitting the code for reviewal."; \
		fi

ci-lint:
	@$(CI_LINT) run --deadline 10m --new-from-rev=HEAD~

bootstrap:
	@$(GO_CMD)  generate -tags tools tools/tools.go