PKG_SPEC=./...
GOTEST=go test
GOTEST_OPT?=-v -race -timeout 30s
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -coverprofile=coverage.txt -covermode=atomic
TOOLS_DIR = ./.tools
FIX=--fix

.DEFAULT_GOAL := precommit

$(TOOLS_DIR)/golangci-lint: go.mod go.sum tools.go
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: precommit
precommit: lint

.PHONY: coverage
coverage:
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)
	go tool cover -html=coverage.txt -o coverage.html

.PHONY: travis-ci
travis-ci: override FIX = 
travis-ci: precommit test coverage

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(PKG_SPEC)

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint
	$(TOOLS_DIR)/golangci-lint run $(FIX)