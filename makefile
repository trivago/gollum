.PHONY: all clean docker docker-dev install freebsd linux mac pi win examples current vendor test unit coverprofile integration example pre-commit vet lint fmt fmt-check ineffassign
.DEFAULT_GOAL := current

VERSION=0.5.0
BUILD_ENV=GORACE="halt_on_error=0"
BUILD_FLAGS=-ldflags=-s

UNIT_TEST_TAGS="unit"
INTEGRATION_TEST_TAGS="integration"

GOFMT_OPTIONS=-s

UNIT_TEST_ONLY_PKGS=$(shell go list -tags ${UNIT_TEST_TAGS} ./... | grep -v "/vendor/" | grep -v "/contrib/" | grep -v "/testing/integration" )
INTEGRATION_TEST_ONLY_PKGS=$(shell go list -tags ${INTEGRATION_TEST_TAGS} ./testing/integration/...)

CHECK_PKGS=$(shell go list ./... | grep -vE '^github.com/trivago/gollum/vendor/')
CHECK_FILES=$(shell find . -type f -name '*.go' | grep -vE '^\./vendor/')

LINT_PKGS=$(shell go list ./... | grep -vE '^github.com/trivago/gollum/(core$$|vendor/)')
LINT_FILES_CORE=$(shell find core -maxdepth 1 -type f -name '*.go' -not -name '*.pb.go')

all: clean vendor test freebsd linux docker mac pi win examples current

#############################################################################################################
# Build targets

freebsd:
	@echo "Building for FreeBSD/x64"
	@GOOS=freebsd GOARCH=amd64 $(BUILD_ENV) go build $(BUILD_FLAGS) -o gollum
	@rm -f dist/gollum-$(VERSION)-FreeBSD_x64.zip
	@zip dist/gollum-$(VERSION)-FreeBSD_x64.zip gollum

linux:
	@echo "Building for Linux/x64"
	@GOOS=linux GOARCH=amd64 $(BUILD_ENV) go build $(BUILD_FLAGS) -o gollum
	@rm -f dist/gollum-$(VERSION)-Linux_x64.zip
	@zip dist/gollum-$(VERSION)-Linux_x64.zip gollum

mac:
	@echo "Building for MacOS X (MacOS/x64)"
	@GOOS=darwin GOARCH=amd64 $(BUILD_ENV) go build $(BUILD_FLAGS) -o gollum
	@rm -f dist/gollum-$(VERSION)-MacOS_x64.zip
	@zip dist/gollum-$(VERSION)-MacOS_x64.zip gollum

pi:
	@echo "Building for Raspberry Pi (Linux/ARMv6)"
	@GOOS=linux GOARCH=arm GOARM=6 $(BUILD_ENV) go build $(BUILD_FLAGS) -o gollum
	@rm -f dist/gollum-$(VERSION)-Linux_Arm6.zip
	@zip dist/gollum-$(VERSION)-Linux_Arm6.zip gollum

win:
	@echo "Building for Windows/x64"
	@GOOS=windows GOARCH=amd64 $(BUILD_ENV) go build $(BUILD_FLAGS) -o gollum.exe
	@rm -f dist/gollum-$(VERSION)-Windows_x64.zip
	@zip dist/gollum-$(VERSION)-Windows_x64.zip gollum

current:
	@$(BUILD_ENV) go build $(BUILD_FLAGS)

install: current
	@go install

examples:
	@echo "Building Examples"
	@zip -j dist/gollum-$(VERSION)-Examples.zip config/*.conf

docker: linux
	@echo "Building docker image"
	@docker build --squash -t trivago/gollum:$(VERSION) .

docker-dev:
	@echo "Building development docker image"
	@docker build -t trivago/gollum:$(VERSION)-dev -f Dockerfile-dev .

#############################################################################################################
# Administrivia

clean:
	@rm -f ./gollum
	@rm -f ./dist/gollum_*.zip
	@go clean

listsrc:
	@find . -mindepth 1 -maxdepth 1 -not -name vendor -not -name debug -not -name docs -not -name .git -not -name .idea

# Lists directories / files present in the filesystem which are ignored by git (via .gitignore et al). For
# completely ignored subtrees, only the topmost directory is listed.
list-gitignored:
	@sh -c 'function find_ignores_rec() { find "$$1" -mindepth 1 -maxdepth 1 -not -name .git | while read e ; do if git check-ignore --quiet "$$e" ; then echo "$$e" ; elif [ -d "$$e" ]; then find_ignores_rec "$$e" ; fi ; done ; }; find_ignores_rec .'

# Recursive make implementation - slow!
#LISTIGNORED_ROOT = .
#list-gitignored:
#	@find $(LISTIGNORED_ROOT) -mindepth 1 -maxdepth 1 -not -name .git | while read e ; do if git check-ignore --quiet "$$e" ; then echo "$$e" ; elif [ -d "$$e" ]; then make list-gitignored LISTIGNORED_ROOT="$$e" ; fi ; done

#############################################################################################################
# Vendor management

vendor:
	@glide cc
	glide update --strip-vendor

# Runs "glide install" in a managed way - clears glide's cache and removes git-ignored stuff from ./vendor.
# This leaves ./vendor in the same state it would be when checked out with git.
vendor-install: _vendor-install vendor-rm-ignored

_vendor-install:
	glide cache-clear
	rm -rf vendor
	glide install --strip-vendor

# Runs "glide update" in a managed way - clears glide's cache and removes ignored stuff from ./vendor
# This leaves ./vendor in the same state it would be when checked out with git.
vendor-update: _vendor-update vendor-rm-ignored

_vendor-update:
	glide cache-clear
	rm -rf vendor
	rm glide.lock
	glide update --strip-vendor

# Lists files & directories under ./vendor that are ignored by git.
vendor-list-ignored:
	@find vendor | git check-ignore --stdin || true

# Removes files & directories under ./vendor that are ignored by git
vendor-rm-ignored:
	$(MAKE) vendor-list-ignored |  while read f ; do rm -vrf "$$f" ; done

#############################################################################################################
# Tests

test: vet lint fmt-check unit integration

unit:
	@echo "go tests SDK"
	$(BUILD_ENV) go test $(BUILD_FLAGS) -v -cover -timeout 10s -race -tags ${UNIT_TEST_TAGS} $(UNIT_TEST_ONLY_PKGS)

coverprofile:
	@echo "go tests -covermode=count -coverprofile=profile.cov"
	@$(BUILD_ENV) go test $(BUILD_FLAGS) -covermode=count -coverprofile=core.cov ./core
	@$(BUILD_ENV) go test $(BUILD_FLAGS) -covermode=count -coverprofile=format.cov ./format
	@$(BUILD_ENV) go test $(BUILD_FLAGS) -covermode=count -coverprofile=filter.cov ./filter
	@$(BUILD_ENV) go test $(BUILD_FLAGS) -covermode=count -coverprofile=router.cov ./router
	@echo "mode: count" > profile.cov
	@cat ./*.cov | grep -v "mode: " >> profile.cov
	@rm core.cov format.cov filter.cov router.cov

integration: current
	@echo "go tests integration"
	$(BUILD_ENV) go test $(BUILD_FLAGS) -v -race -tags="integration" $(INTEGRATION_TEST_ONLY_PKGS)

pre-commit: vet lint fmt

vet:
	@echo "Running go vet"
	@go vet $(CHECK_PKGS)

lint:
	@echo "Running golint against core"
	@golint -set_exit_status $(LINT_FILES_CORE)
	@echo "Running golint against other packages"
	@golint -set_exit_status $(LINT_PKGS)

fmt:
	@echo "Running go fmt"
	@go fmt $(CHECK_PKGS)

fmt-list:
	@gofmt $(GOFMT_OPTIONS) -l $(CHECK_FILES)

fmt-diff:
	@gofmt $(GOFMT_OPTIONS) -d $(CHECK_FILES)

fmt-check:
ifneq ($(shell gofmt $(GOFMT_OPTIONS) -l $(CHECK_FILES) | wc -l | tr -d "[:blank:]"), 0)
	$(error gofmt returns more than one line, run 'make fmt-check' or 'make fmt-diff' for details, 'make fmt' to fix)
endif
	@echo "gofmt check successful"

ineffassign:
	@echo "Running ineffassign"
	@go get -u github.com/gordonklaus/ineffassign
	@ineffassign ./

# .git/hooks/pre-commit
#
# #!/bin/bash
# # Run tests
# make vet lint fmt-check ineffassign >&2
# exit $?

