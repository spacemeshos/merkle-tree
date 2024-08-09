GOLANGCI_LINT_VERSION := v1.59.0
GOTESTSUM_VERSION := v1.12.0

all: build test
.PHONY: all

build:
	go build -v ./...
.PHONY: build

test:
	gotestsum -- -timeout 5m -p 1 -race ./...
.PHONY: test

install:
	go mod download
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)
	go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
.PHONY: install

tidy:
	go mod tidy
.PHONY: tidy

test-tidy:
	# Working directory must be clean, or this test would be destructive
	git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go mod tidy` not to change anything, the test should fail otherwise
	make tidy
	git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-tidy

test-fmt:
	git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go fmt` not to change anything, the test should fail otherwise
	go fmt ./...
	git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-fmt

clear-test-cache:
	go clean -testcache
.PHONY: clear-test-cache

lint:
	go vet ./...
	./bin/golangci-lint run --config .golangci.yml
.PHONY: lint

# Auto-fixes golangci-lint issues where possible.
lint-fix:
	./bin/golangci-lint run --config .golangci.yml --fix
.PHONY: lint-fix

lint-github-action:
	go vet ./...
	./bin/golangci-lint run --config .golangci.yml --out-format=github-actions
.PHONY: lint-github-action

cover:
	go test -coverprofile=cover.out -timeout 0 -p 1 -coverpkg=./... $(UNIT_TESTS)
.PHONY: cover
