.DEFAULT_GOAL := help

GOMOD ?= on

GOLLUM_TAG := $(shell git describe --always --tags --match "v*" | sed -E 's/^v([0-9\.]+)(-[0-9]+){0,1}((-)g([a-f0-9]+)){0,1}.*/\1\2\4\5/')
GOLLUM_RELEASE_SUFFIX := # Can be set for custom releases to also include plugin versions
GOLLUM_DIRTY := $(if $(shell git status --porcelain),-dirty)
GOLLUM_VERSION := $(GOLLUM_TAG)$(GOLLUM_DIRTY)$(if $(GOLLUM_RELEASE_SUFFIX),+$(GOLLUM_RELEASE_SUFFIX))

GO_ENV := GORACE="halt_on_error=0" GO111MODULE="$(GOMOD)"
GO_FLAGS := -ldflags="-s -X 'github.com/trivago/gollum/core.versionString=$(GOLLUM_VERSION)'"
GO_FLAGS_DEBUG := $(GO_FLAGS) -ldflags='-linkmode=internal' -gcflags='-N -l'

TAGS_GOLLUM=netgo
TAGS_UNIT=unit
TAGS_INTEGRATION=integration

#############################################################################################################
# Build related targets

.PHONY: freebsd # Build gollum zip-file for FreeBSD (x64)
freebsd:
	@echo "\033[0;33mBuilding for FreeBSD/x64\033[0;0m"
	@GOOS=freebsd GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@zip dist/gollum-$(GOLLUM_VERSION)-FreeBSD_x64.zip gollum

.PHONY: linux # Build gollum zip-file for Linux (x64)
linux:
	@echo "\033[0;33mBuilding for Linux/x64\033[0;0m"
	@GOOS=linux GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@zip dist/gollum-$(GOLLUM_VERSION)-Linux_x64.zip gollum

.PHONY: mac # Build gollum zip-file for MacOS X (x64)
mac:
	@echo "\033[0;33mBuilding for MacOS X (MacOS/x64)\033[0;0m"
	@GOOS=darwin GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@zip dist/gollum-$(GOLLUM_VERSION)-MacOS_x64.zip gollum

.PHONY: pi # Build gollum zip-file for Raspberry Pi / Linux (ARMv6)
pi:
	@echo "\033[0;33mBuilding for Raspberry Pi (Linux/ARMv6)\033[0;0m"
	@GOOS=linux GOARCH=arm GOARM=6 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@zip dist/gollum-$(GOLLUM_VERSION)-Linux_Arm6.zip gollum

.PHONY: win # Build gollum zip-file for Windows (x64)
win:
	@echo "\033[0;33mBuilding for Windows/x64\033[0;0m"
	@GOOS=windows GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum.exe
	@zip dist/gollum-$(GOLLUM_VERSION)-Windows_x64.zip gollum.exe

.PHONY: docker # Build the gollum docker image
docker:
	@echo "\033[0;33mBuilding docker image\033[0;0m"
	@docker build -t trivago/gollum:$(GOLLUM_VERSION) .

.PHONY: docker-dev # Build the gollum docker development image
docker-dev:
	@echo "\033[0;33mBuilding development docker image\033[0;0m"
	@docker build -t trivago/gollum:$(GOLLUM_VERSION)-dev -f Dockerfile-dev .

.PHONY: build # Build the gollum binary for the current platform
build:
	@$(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)"

.PHONY: debug # Build the gollum binary for the current platform with additional debugging flags 
debug:
	@$(GO_ENV) go build $(GO_FLAGS_DEBUG) -tags="$(TAGS_GOLLUM)"

.PHONY: all # Test and build all distributions of gollum
all: clean test freebsd linux docker mac pi win

.PHONY: install # Install the current gollum build to the system
install: build
	@go install

.PHONY: clean # Remove all files created by build and distribution targets
clean:
	@rm -f ./gollum
	@rm -f ./dist/gollum-*.zip
	@go clean

#############################################################################################################
# Vendor related targets

.PHONY: vendor # Generate the vendor folder
vendor:
	@$(GO_ENV) go mod vendor

.PHONY: vendor-clean # Removes files & directories under ./vendor that are ignored by git
vendor-clean:
	@find vendor | git check-ignore --stdin | while read f ; do rm -vrf "$$f" ; done

#############################################################################################################
# Test related targets

.PHONY: test # Run all relevant tests (no native tests)
test: test-unit test-integration

.PHONY: test-unit # Run all unit tests
test-unit:
	@echo "\033[0;33mRunning unit tests\033[0;0m"
	@$(GO_ENV) go test $(GO_FLAGS) -v -cover -timeout 10s -race -tags="$(TAGS_GOLLUM) $(TAGS_UNIT)" ./...

.PHONY: test-native # Run all unit tests for native plugins
test-native:
	@echo "\033[0;33mRunning unit tests for native plugins\033[0;0m"
	@$(GO_ENV) go test $(GO_FLAGS) -v -cover -timeout 10s -race -tags="$(TAGS_GOLLUM)" ./contrib/native/...

.PHONY: test-integration # Run all integration tests
test-integration:: build
	@echo "\033[0;33mRunning integration tests\033[0;0m"
	@$(GO_ENV) go test $(GO_FLAGS) -v -race -tags="$(TAGS_GOLLUM) $(TAGS_INTEGRATION)" ./testing/integration

#############################################################################################################
# Linter related targets

.PHONY: lint # Run all linters
lint: lint-fmt lint-meta

.PHONY: lint-meta # Run the go meta linter
lint-meta:
	@echo "\033[0;33mRunning go linters\033[0;0m"
	@gometalinter --vendor --cyclo-over=20 \
	--disable=goconst \
	--disable=gas \
	--disable=maligned \
	--disable=gocyclo \
	--disable=errcheck \
	--disable=gosec \
	--disable=gotype \
	--exclude="\.[cC]lose[^ ]*\(.*\) \(errcheck\)" \
	--exclude="\/go[\/]?1\.[0-9]{1,2}\." \
	--exclude="^vendor\/" \
	--skip=contrib \
	--skip=docs \
	--skip=testing \
	--deadline=5m \
	--concurrency=4 \
	 ./...
	@echo "\033[0;32mDone\033[0;0m"

.PHONY: lint-fmt # Run go fmt and see if anything would be changed
lint-fmt:
	@echo "\033[0;33mRunning go fmt\033[0;0m"
ifneq ($(shell go list -f '"cd {{.Dir}}; gofmt -s -l {{join .GoFiles " "}}"' ./... | xargs sh -c), )
	@go list -f '"cd {{.Dir}}; gofmt -s -l {{join .GoFiles " "}}"' ./... | xargs sh -c
	@echo "\033[0;31mFAILED\033[0;0m"
	@exit 1
else
	@echo "\033[0;32mOK\033[0;0m"
endif

#############################################################################################################
# Build pipeline targets

.PHONY: pipeline-tools # Go get required tools
pipeline-tools:
	@echo "\033[0;33mInstalling required go tools ...\033[0;0m"
	@$(GO_ENV) go get -u github.com/mattn/goveralls
	@cd $$GOPATH; curl -L https://git.io/vp6lP | sh
	@echo "\033[0;32mDone\033[0;0m"

.PHONY: pipeline-accept # Accept runs all targets required for PR acceptance
pipeline-accept: | lint test

.PHONY: pipeline-build # Run all relevant platform builds required for PR acceptance
pipeline-build: mac linux win freebsd

.PHONY: pipeline-coverage # Generate cover profiles files required for PR acceptance
pipeline-coverage:
	@echo "\033[0;33mGenerating cover profiles\033[0;0m"
	@$(GO_ENV) go test $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -covermode=count -coverprofile=core.cov ./core
	@$(GO_ENV) go test $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -covermode=count -coverprofile=format.cov ./format
	@$(GO_ENV) go test $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -covermode=count -coverprofile=filter.cov ./filter
	@$(GO_ENV) go test $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -covermode=count -coverprofile=router.cov ./router

	@echo "INFO: start generating profile.cov"
	@rm -f profile.tmp profile.cov

	@echo "mode: count" > profile.tmp
	@for cov_file in  $$( ls -f *.cov ); do cat $${cov_file} | grep -v "mode: " >> profile.tmp ; done
	@mv profile.tmp profile.cov

	@echo "INFO: profile.cov successfully generated"
	@rm core.cov format.cov filter.cov router.cov

#############################################################################################################

.PHONY: help # Print the make help screen
help:
	@echo "GOLLUM v$(GOLLUM_VERSION)"
	@echo "Make targets overview"
	@echo
	@echo "\033[0;33mAvailable targets:"
	@grep '^.PHONY: .* #' makefile | sed -E 's/\.PHONY: (.*) # (.*)/"\1" "\2"/g' | xargs printf "  \033[0;32m%-25s \033[0;0m%s\n"
