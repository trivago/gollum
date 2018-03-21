.DEFAULT_GOAL := current

GOLLUM_TAG := $(shell git describe --tags --match "v*" | sed -E 's/^v([0-9\.]+)(-[0-9]+){0,1}((-)g([a-f0-9]+)){0,1}.*/\1\2\4\5/')
GOLLUM_DIRTY := $(if $(shell git status --porcelain),-dirty)
GOLLUM_VERSION := $(join $(GOLLUM_TAG),$(GOLLUM_DIRTY))

GO_ENV := GORACE="halt_on_error=0"
GO_FLAGS := -ldflags="-s -X 'github.com/trivago/gollum/core.versionString=$(GOLLUM_VERSION)'"
GO_FLAGS_DEBUG := $(GO_FLAGS) -ldflags='-linkmode=internal' -gcflags='-N -l'

GO_TAGS=netgo
UNIT_TEST_TAG=unit
INTEGRATION_TEST_TAG=integration

GOFMT_OPTIONS=-s

UNIT_TEST_ONLY_PKGS=$(shell go list -tags="${UNIT_TEST_TAG}" ./... | grep -v "/vendor/" | grep -v "/contrib/" | grep -v "/testing/integration" )
INTEGRATION_TEST_ONLY_PKGS=$(shell go list -tags="${INTEGRATION_TEST_TAG}" ./testing/integration/...)

CHECK_PKGS=$(shell go list ./... | grep -vE '^github.com/trivago/gollum/vendor/')
CHECK_FILES=$(shell find . -type f -name '*.go' | grep -vE '^\./vendor/')

LINT_PKGS=$(shell go list ./... | grep -vE '^github.com/trivago/gollum/(core$$|vendor/)')
LINT_FILES_CORE=$(shell find core -maxdepth 1 -type f -name '*.go' -not -name '*.pb.go')

LS_COV=ls -f *.cov

#############################################################################################################
# Build targets

all:: clean test freebsd linux docker mac pi win

freebsd::
	@echo "Building for FreeBSD/x64"
	@GOOS=freebsd GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="${GO_TAGS}" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-FreeBSD_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-FreeBSD_x64.zip gollum

linux::
	@echo "Building for Linux/x64"
	@GOOS=linux GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="${GO_TAGS}" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-Linux_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-Linux_x64.zip gollum

mac::
	@echo "Building for MacOS X (MacOS/x64)"
	@GOOS=darwin GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="${GO_TAGS}" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-MacOS_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-MacOS_x64.zip gollum

pi::
	@echo "Building for Raspberry Pi (Linux/ARMv6)"
	@GOOS=linux GOARCH=arm GOARM=6 $(GO_ENV) go build $(GO_FLAGS) -tags="${GO_TAGS}" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-Linux_Arm6.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-Linux_Arm6.zip gollum

win::
	@echo "Building for Windows/x64"
	@GOOS=windows GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="${GO_TAGS}" -o gollum.exe
	@rm -f dist/gollum-$(GOLLUM_VERSION)-Windows_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-Windows_x64.zip gollum

current::
	@$(GO_ENV) go build $(GO_FLAGS) -tags="${GO_TAGS}"

debug::
	@$(GO_ENV) go build $(GO_FLAGS_DEBUG) -tags="${GO_TAGS}"

install:: current
	@go install

docker:: linux
	@echo "Building Docker Alpine image"
	@docker build -f docker/apline.docker --squash -t trivago/gollum:$(GOLLUM_VERSION) .

docker-prod::
	@echo "Building Docker Debian production image"
	@docker build --build-arg 'VERSION=$(GOLLUM_VERSION)' -f docker/debian.docker -t trivago/gollum:$(GOLLUM_VERSION)-debian-stretch .

docker-dev::
	@echo "Building Docker development image"
	@docker build -f docker/dev.docker -t trivago/gollum:$(GOLLUM_VERSION)-dev .

#############################################################################################################
# Administrivia

clean::
	@rm -f ./gollum
	@rm -f ./dist/gollum_*.zip
	@go clean

list-src::
	@find . -mindepth 1 -maxdepth 1 -not -name vendor -not -name debug -not -name docs -not -name .git -not -name .idea

# Lists directories / files present in the filesystem which are ignored by git (via .gitignore et al). For
# completely ignored subtrees, only the topmost directory is listed.
list-gitignored::
	@sh -c 'function find_ignores_rec() {\
	find "$$1" -mindepth 1 -maxdepth 1 -not -name .git |\
		while read e ; do\
			if git check-ignore --quiet "$$e"; then\
				echo "$$e";\
			elif [ -d "$$e" ]; then\
				find_ignores_rec "$$e";\
			fi;\
		done;\
	}; \
	find_ignores_rec .'

#############################################################################################################
# Vendor management

vendor:
	@dep init

vendor-update:: vendor
	@dep ensure

# Lists files & directories under ./vendor that are ignored by git.
vendor-list-ignored::
	@find vendor | git check-ignore --stdin || true

# Removes files & directories under ./vendor that are ignored by git
vendor-rm-ignored::
	$(MAKE) vendor-list-ignored |  while read f ; do rm -vrf "$$f" ; done

#############################################################################################################
# Tests

test:: vet lint fmt-check unit integration

unit::
	@echo "go tests SDK"
	$(GO_ENV) go test $(GO_FLAGS) -v -cover -timeout 10s -race -tags="${GO_TAGS} $(UNIT_TEST_TAG)" $(UNIT_TEST_ONLY_PKGS)

coverprofile::
	@echo "go tests -covermode=count -coverprofile=profile.cov"
	@$(GO_ENV) go test $(GO_FLAGS) -tags="${GO_TAGS}" -covermode=count -coverprofile=core.cov ./core
	@$(GO_ENV) go test $(GO_FLAGS) -tags="${GO_TAGS}" -covermode=count -coverprofile=format.cov ./format
	@$(GO_ENV) go test $(GO_FLAGS) -tags="${GO_TAGS}" -covermode=count -coverprofile=filter.cov ./filter
	@$(GO_ENV) go test $(GO_FLAGS) -tags="${GO_TAGS}" -covermode=count -coverprofile=router.cov ./router

	@echo "INFO: start generating profile.cov"
	@rm -f profile.tmp profile.cov

	@echo "mode: count" > profile.tmp
	@for cov_file in  $$( $(LS_COV) ); do cat $${cov_file} | grep -v "mode: " >> profile.tmp ; done
	@mv profile.tmp profile.cov

	@echo "INFO: profile.cov successfully generated"
	@rm core.cov format.cov filter.cov router.cov

integration:: current
	@echo "go tests integration"
	$(GO_ENV) go test $(GO_FLAGS) -v -race -tags="${GO_TAGS} $(INTEGRATION_TEST_TAG)" $(INTEGRATION_TEST_ONLY_PKGS)

pre-commit:: vet lint fmt ineffassign

vet::
	@echo "Running go vet"
	@go vet $(CHECK_PKGS)

lint::
	@echo "Running golint against core"
	@golint -set_exit_status $(LINT_FILES_CORE)
	@echo "Running golint against other packages"
	@golint -set_exit_status $(LINT_PKGS)

fmt::
	@echo "Running go fmt"
	@go fmt $(CHECK_PKGS)

fmt-list::
	@gofmt $(GOFMT_OPTIONS) -l $(CHECK_FILES)

fmt-diff::
	@gofmt $(GOFMT_OPTIONS) -d $(CHECK_FILES)

fmt-check::
ifneq ($(shell gofmt $(GOFMT_OPTIONS) -l $(CHECK_FILES) | wc -l | tr -d "[:blank:]"), 0)
	$(error gofmt returns more than one line, run 'make fmt-check' or 'make fmt-diff' for details, 'make fmt' to fix)
endif
	@echo "gofmt check successful"

ineffassign::
	@echo "Running ineffassign"
	@go get -u github.com/gordonklaus/ineffassign
	@ineffassign ./

# .git/hooks/pre-commit
#
# #!/bin/bash
# # Run tests
# make pre-commit
# exit $?

