# Set .SHELLFLAGS to use "bash strict mode". This will fail on any error, fail
# on an error in a pipeline (usually it just returns the value of the last
# command in a pipeline), and will fail if using any undefined variables.
SHELL := bash
# Set .ONESHELL config. Runs the whole make recipe in one shell session.
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
# Both of the above are GNU Make 3.82+. macOS ships 3.81 and ignores them,
# so a multi-command recipe there gets no strict mode and needs its own
# `set -e` to keep a failure from passing as success.
# Set .DELETE_ON_ERROR. Deletes any files generated on error.
.DELETE_ON_ERROR:
# Set makeflags to --warn-undefined-variables (probably an error) and to avoid
# the built-in rules (this removes a lot of magic that is related to yacc and
# other tools).
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# frame is cgo-free on every platform, enforced by exporting this to every
# recipe: an accidental cgo dependency fails the build instead of sneaking
# in. On darwin, capture will use the system `screencapture` binary via
# os/exec -- never a cgo backend.
export CGO_ENABLED := 0

# VARIABLES
GOSERVICES = $(sort $(notdir $(realpath $(dir $(wildcard ./cmd/*/main.go)))))

.PHONY: $(GOSERVICES)

.PHONY: build
build: $(GOSERVICES)

# creates a TARGET per go service. Build by package path, not file list --
# Go only embeds VCS info (used by `frame version`) for package builds.
$(GOSERVICES): % : ./cmd/%/main.go
	go build -o bin/$@ ./cmd/$@

# list available go services
.PHONY: services
services:
	@echo $(GOSERVICES)

.PHONY: test
test:
	go test ./...

# platforms `cross` proves the build on. Each is a target of its own too, so
# `make darwin/arm64` checks just that one.
GOCROSSPLATFORMS = linux/amd64 windows/amd64 darwin/arm64

# prove every service builds cgo-free for every shipping platform.
.PHONY: cross $(GOCROSSPLATFORMS)
cross: $(GOCROSSPLATFORMS)

$(GOCROSSPLATFORMS):
	GOOS=$(@D) GOARCH=$(@F) go build -o /dev/null ./cmd/...
