.DEFAULT_GOAL := current

GOLLUM_TAG := $(shell git describe --tags --match "v*" | sed -E 's/^v([0-9\.]+)(-[0-9]+){0,1}((-)g([a-f0-9]+)){0,1}.*/\1\2\4\5/')
GOLLUM_DIRTY := $(if $(shell git status --porcelain),-dirty)
GOLLUM_VERSION := $(join $(GOLLUM_TAG),$(GOLLUM_DIRTY))

GO_ENV := GORACE="halt_on_error=0"
GO_FLAGS := -ldflags="-s -X 'github.com/trivago/gollum/core.versionString=$(GOLLUM_VERSION)'"
GO_FLAGS_DEBUG := $(GO_FLAGS) -ldflags='-linkmode=internal' -gcflags='-N -l'

TAGS_GOLLUM=netgo
TAGS_UNIT=unit
TAGS_INTEGRATION=integration

#############################################################################################################
# Build related targets

.PHONY: all # Test and build all distributions of gollum
all: clean test freebsd linux docker mac pi win

.PHONY: freebsd # Build gollum zip-file for FreeBSD (x64)
freebsd:
	@echo "Building for FreeBSD/x64"
	@GOOS=freebsd GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-FreeBSD_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-FreeBSD_x64.zip gollum

.PHONY: linux # Build gollum zip-file for Linux (x64)
linux:
	@echo "Building for Linux/x64"
	@GOOS=linux GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-Linux_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-Linux_x64.zip gollum

.PHONY: mac # Build gollum zip-file for MacOS X (x64)
mac:
	@echo "Building for MacOS X (MacOS/x64)"
	@GOOS=darwin GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-MacOS_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-MacOS_x64.zip gollum

.PHONY: pi # Build gollum zip-file for Raspberry Pi / Linux (ARMv6)
pi:
	@echo "Building for Raspberry Pi (Linux/ARMv6)"
	@GOOS=linux GOARCH=arm GOARM=6 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum
	@rm -f dist/gollum-$(GOLLUM_VERSION)-Linux_Arm6.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-Linux_Arm6.zip gollum

.PHONY: win # Build gollum zip-file for Windows (x64)
win:
	@echo "Building for Windows/x64"
	@GOOS=windows GOARCH=amd64 $(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)" -o gollum.exe
	@rm -f dist/gollum-$(GOLLUM_VERSION)-Windows_x64.zip
	@zip dist/gollum-$(GOLLUM_VERSION)-Windows_x64.zip gollum

.PHONY: docker # Build the gollum docker image
docker: linux
	@echo "Building docker image"
	@docker build --squash -t trivago/gollum:$(GOLLUM_VERSION) .

.PHONY: docker-dev # Build the gollum docker development image
docker-dev:
	@echo "Building development docker image"
	@docker build -t trivago/gollum:$(GOLLUM_VERSION)-dev -f Dockerfile-dev .

.PHONY: current # Build the gollum binary for the current platform
current:
	@$(GO_ENV) go build $(GO_FLAGS) -tags="$(TAGS_GOLLUM)"

.PHONY: debug # Build the gollum binary for the current platform with additional debugging flags 
debug:
	@$(GO_ENV) go build $(GO_FLAGS_DEBUG) -tags="$(TAGS_GOLLUM)"

.PHONY: install # Install the current gollum build to the system
install: current
	@go install

.PHONY: clean # Remove all files created by build and distribution targets
clean:
	@rm -f ./gollum
	@rm -f ./dist/gollum_*.zip
	@go clean

#############################################################################################################
# Vendor related targets

Gopkg.toml:
	@dep init

Gopkg.lock: Gopkg.toml
	@dep ensure

.PHONY: vendor # Generate the vendor folder
vendor: Gopkg.toml
	@dep ensure

.PHONY: vendor # Update all dependencies in the vendor folder
vendor-update: Gopkg.lock
	@dep ensure -update

.PHONY: vendor-clean # Removes files & directories under ./vendor that are ignored by git
vendor-clean:
	find vendor | git check-ignore --stdin | while read f ; do rm -vrf "$$f" ; done

#############################################################################################################
# Test related targets

.PHONY: test # Run all tests
test: test-unit test-integration

.PHONY: test-unit # Run all unit tests
test-unit:
	@echo "go tests SDK"
	$(GO_ENV) go test $(GO_FLAGS) -v -cover -timeout 10s -race -tags="$(TAGS_GOLLUM) $(TAGS_UNIT)" ./...

.PHONY: test-integration # Run all integration tests
test-integration:: current
	@echo "go tests integration"
	$(GO_ENV) go test $(GO_FLAGS) -v -race -tags="$(TAGS_GOLLUM) $(TAGS_INTEGRATION)" ./...

#############################################################################################################
# Linter related targets

.PHONY: lint # Run all linters
lint: lint-fmt
	@gometalinter.v2 --vendor \
	--disable=gocyclo \
	--disable=goconst \
	--disable=deadcode \
	--disable=gas \
	--disable=structcheck \
	--disable=errcheck \
	--disable=maligned \
	--disable=interfacer \
	--skip=contrib \
	--skip=docs \
	--skip=testing \
	 ./...

.PHONY: lint-fmt # Run go fmt and see if anything would be changed
lint-fmt:
	@echo "Running go fmt"
ifneq ($(shell go list -f '"cd {{.Dir}}; gofmt -s -l {{join .GoFiles " "}}"' ./... | xargs sh -c), )
	@go list -f '"cd {{.Dir}}; gofmt -s -l {{join .GoFiles " "}}"' ./... | xargs sh -c
	@echo "FAILED"
	@exit 1
else
	@echo "OK"
endif

#############################################################################################################
# Build pipeline targets

.PHONY: install-tools # Go get required tools
install-tools:
	@go get github.com/mattn/goveralls
	@go get gopkg.in/alecthomas/gometalinter.v2
	@gometalinter.v2 --install

.PHONY: accept # Accept runs all targets required for PR acceptance
accept: lint test

.PHONY: build # Run all platform builds required for PR acceptance
build: mac linux win

.PHONY: coverprofile # Generate coveralls profile files required for PR acceptance
coverprofile:
	@echo "go tests -covermode=count -coverprofile=profile.cov"
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