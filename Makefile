# TOOLCHAIN
GO	  := CGO_ENABLED=0 go
CGO	  := CGO_ENABLED=1 go
GOFMT := $(GO)fmt

# ENVIRONMENT
VERBOSE	=
GOPATH	:= $(GOPATH)

# APPLICATION INFORMATION
BUILD_DATE	:= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
REVISION	:= $(shell git rev-parse --short HEAD)
RELEASE		:= $(shell git describe --tags 2>/dev/null || git rev-parse --short HEAD)-dev
USER		:= $(shell whoami)

# GO TOOLS
GOTOOLS := $(shell cat tools.go | grep "_ \"" | awk '{ print $$2 }' | tr -d '"')

# MISC
COVERPROFILE	:= coverage.out
DIST_DIR		:= dist

# GO TAGS
GO_TAGS := osusergo netgo static_build

# GO LD FLAGS
GO_LD_FLAGS := -s -w -extldflags "-fno-PIC -static -Wl -z now -z relro"
GO_LD_FLAGS += -X github.com/axiomhq/pkg/version.release=$(RELEASE)
GO_LD_FLAGS += -X github.com/axiomhq/pkg/version.revision=$(REVISION)
GO_LD_FLAGS += -X github.com/axiomhq/pkg/version.buildDate=$(BUILD_DATE)
GO_LD_FLAGS += -X github.com/axiomhq/pkg/version.buildUser=$(USER)

# FLAGS
GO_FLAGS 			:= -buildvcs=false -buildmode=pie -installsuffix=cgo -trimpath -tags='$(GO_TAGS)' -ldflags='$(GO_LD_FLAGS)'
GO_TEST_FLAGS		:= -race -coverprofile=$(COVERPROFILE)
GORELEASER_FLAGS	:= --snapshot --clean

# DEPENDENCIES
GOMODDEPS = go.mod go.sum

# Enable verbose test output if explicitly set.
GOTESTSUM_FLAGS	=
ifdef VERBOSE
	GOTESTSUM_FLAGS += --format=standard-verbose
endif

# FUNCTIONS
# func go-run-tool(name)
go-run-tool = $(CGO) run $(shell echo $(GOTOOLS) | tr ' ' '\n' | grep -w $1)
# func go-list-pkg-sources(package)
go-list-pkg-sources = $(GO) list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $(1)
# func go-pkg-sourcefiles(package)
go-pkg-sourcefiles = $(shell $(call go-list-pkg-sources,$(strip $1)))

.PHONY: all
all: dep fmt lint test build ## Run dep, fmt, lint, test and build

.PHONY: build
build: dep.stamp $(call go-pkg-sourcefiles, ./...) ## Build the binaries
	@echo ">> building binaries"
	@$(call go-run-tool, goreleaser) build $(GORELEASER_FLAGS)

.PHONY: clean
clean: ## Remove build and test artifacts
	@echo ">> cleaning up artifacts"
	@rm -rf bin $(DIST_DIR) $(COVERPROFILE) dep.stamp

.PHONY: cover
cover: $(COVERPROFILE) ## Calculate the code coverage score
	@echo ">> calculating code coverage"
	@$(GO) tool cover -func=$(COVERPROFILE) | tail -n1

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
dep-upgrade-tools: ## Upgrade all tool dependencies to their latest version
	@echo ">> upgrading tool dependencies"
	@$(GO) get -d $(GOTOOLS)
	@make dep

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
lint: ## Lint the source code
	@echo ">> linting code"
	@$(call go-run-tool, golangci-lint) run

.PHONY: test
test: ## Run all tests. Run with VERBOSE=1 to get verbose test output ('-v' flag).
	@echo ">> running tests"
	@$(call go-run-tool, gotestsum) $(GOTESTSUM_FLAGS) -- $(GO_TEST_FLAGS) ./...

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# MISC TARGETS

$(COVERPROFILE):
	@make test
