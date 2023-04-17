SHELL := bash
.DEFAULT_GOAL := test
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules



.PHONY: test
test: ## - Runs go test with default values
	@printf "\033[32m\xE2\x9c\x93 Testing your code to find potential problems\n\033[0m"
	go test -v -count=1 -trimpath -race ./...
	

.PHONY: lint
lint: ## - Lint the application code for problems and nits
	@printf "\033[32m\xE2\x9c\x93 Linting your code to find potential problems\n\033[0m"
	go vet ./...
	@PATH="${GOPATH}/bin:${PATH}" "${HOME}/go/bin/goimports" -l -w -local github.com/StevenACoffman/ .
	@PATH="${GOPATH}/bin:${PATH}" "${HOME}/go/bin/golines" --shorten-comments --base-formatter="gofumpt" -w .
	@PATH="${GOPATH}/bin:${PATH}" "${HOME}/go/bin/golangci-lint" run --config=.golangci.yaml ./...
.PHONY: help
## help: Prints this help message
help: ## - Show help message
	@printf "\033[32m\xE2\x9c\x93 usage: make [target]\n\n\033[0m"
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

