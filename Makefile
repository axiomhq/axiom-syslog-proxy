# TOOLCHAIN
GO				:= CGO_ENABLED=0 GOBIN=$(CURDIR)/bin go
GO_BIN_IN_PATH  := CGO_ENABLED=0 go
GOFMT			:= $(GO)fmt

# ENVIRONMENT
VERBOSE		=
GOPATH		:= $(GOPATH)

# APPLICATION INFORMATION
BUILD_DATE      := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
REVISION        := $(shell git rev-parse --short HEAD)
RELEASE         := $(shell git describe --tags 2>/dev/null || git rev-parse --short HEAD)-dev
USER            := $(shell whoami)

# GO TOOLS
GOLANGCI_LINT	:= bin/golangci-lint
GORELEASER		:= bin/goreleaser
GOTESTSUM		:= bin/gotestsum

GOTOOLS := $(shell cat tools.go | grep "_ \"" | awk '{ print $$2 }' | tr -d '"')

# MISC
COVERPROFILE	:= coverage.out
DIST_DIR		:= dist

# TAGS
GOTAGS := osusergo netgo static_build

# FLAGS
GOFLAGS := -buildmode=pie -tags='$(GOTAGS)' -installsuffix=cgo -trimpath
GOFLAGS += -ldflags='-s -w -extldflags "-fno-PIC -static -Wl -z now -z relro"
GOFLAGS += -X github.com/axiomhq/pkg/version.release=$(RELEASE)
GOFLAGS += -X github.com/axiomhq/pkg/version.revision=$(REVISION)
GOFLAGS += -X github.com/axiomhq/pkg/version.buildDate=$(BUILD_DATE)
GOFLAGS += -X github.com/axiomhq/pkg/version.buildUser=$(USER)'

GO_TEST_FLAGS		:= -race -coverprofile=$(COVERPROFILE)
GORELEASER_FLAGS	:= --snapshot --rm-dist

# DEPENDENCIES
GOMODDEPS = go.mod go.sum

# Enable verbose test output if explicitly set.
GOTESTSUM_FLAGS	=
ifdef VERBOSE
	GOTESTSUM_FLAGS += --format=standard-verbose
endif

# FUNCTIONS
# func go-list-pkg-sources(package)
go-list-pkg-sources = $(GO) list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $(1)
# func go-pkg-sourcefiles(package)
go-pkg-sourcefiles = $(shell $(call go-list-pkg-sources,$(strip $1)))

.PHONY: all
all: dep fmt lint test build ## Run dep, fmt, lint, test and build

.PHONY: build
build: $(GORELEASER) dep.stamp $(call go-pkg-sourcefiles, ./...) ## Build the binaries
	@echo ">> building binaries"
	@$(GORELEASER) build $(GORELEASER_FLAGS)

.PHONY: clean
clean: ## Remove build and test artifacts
	@echo ">> cleaning up artifacts"
	@rm -rf bin $(DIST_DIR) $(COVERPROFILE) dep.stamp

.PHONY: coverage
coverage: $(COVERPROFILE) ## Calculate the code coverage score
	@echo ">> calculating code coverage"
	@$(GO) tool cover -func=$(COVERPROFILE) | grep total | awk '{print $$3}'

.PHONY: dep-clean
dep-clean: ## Remove obsolete dependencies
	@echo ">> cleaning dependencies"
	@$(GO) mod tidy

.PHONY: dep-upgrade
dep-upgrade: ## Upgrade all direct dependencies to their latest version
	@echo ">> upgrading dependencies"
	@$(GO) get -d $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	@make dep

.PHONY: dep-upgrade-tools
dep-upgrade-tools: $(GOTOOLS) ## Upgrade all tool dependencies to their latest version and install them

.PHONY: dep
dep: dep-clean dep.stamp ## Install and verify dependencies and remove obsolete ones

dep.stamp: $(GOMODDEPS)
	@echo ">> installing dependencies"
	@$(GO) mod download
	@$(GO) mod verify
	@touch $@

.PHONY: fmt
fmt: ## Format and simplify the source code using `gofmt`
	@echo ">> formatting code"
	@! $(GOFMT) -s -w $(shell find . -path -prune -o -name '*.go' -print) | grep '^'

.PHONY: install
install: $(GOPATH)/bin/axiom-syslog-proxy ## Install the binary into the $GOPATH/bin directory

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint the source code
	@echo ">> linting code"
	@$(GOLANGCI_LINT) run

.PHONY: test
test: $(GOTESTSUM) ## Run all tests. Run with VERBOSE=1 to get verbose test output (`-v` flag)
	@echo ">> running tests"
	@$(GOTESTSUM) $(GOTESTSUM_FLAGS) -- $(GO_TEST_FLAGS) ./...

.PHONY: tools
tools: $(GOLANGCI_LINT) $(GORELEASER) $(GOTESTSUM) ## Install all tools into the projects local $GOBIN directory

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# MISC TARGETS

$(COVERPROFILE):
	@make test

# INSTALL TARGETS

$(GOPATH)/bin/axiom-syslog-proxy: dep.stamp $(call go-pkg-sourcefiles, ./...)
	@echo ">> installing axiom-syslog-proxy binary"
	@$(GO_BIN_IN_PATH) install $(GOFLAGS) ./cmd/axiom-syslog-proxy

# GO TOOLS

$(GOLANGCI_LINT): dep.stamp $(call go-pkg-sourcefiles, github.com/golangci/golangci-lint/cmd/golangci-lint)
	@echo ">> installing golangci-lint"
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint

$(GORELEASER): dep.stamp $(call go-pkg-sourcefiles, github.com/goreleaser/goreleaser)
	@echo ">> installing goreleaser"
	@$(GO) install github.com/goreleaser/goreleaser

$(GOTESTSUM): dep.stamp $(call go-pkg-sourcefiles, gotest.tools/gotestsum)
	@echo ">> installing gotestsum"
	@$(GO) install gotest.tools/gotestsum

$(GOTOOLS): dep.stamp $(call go-pkg-sourcefiles, $@)
	@echo ">> installing $@"
	@$(GO) get -d $@
	@$(GO) install $@
