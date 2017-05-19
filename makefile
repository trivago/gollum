.PHONY: all clean docker docker-dev install freebsd linux mac pi win examples current vendor test unit coverprofile integration example pre-commit vet lint fmt
.DEFAULT_GOAL := current

VERSION=0.5.0
BUILD_ENV=GO15VENDOREXPERIMENT=1 GORACE="halt_on_error=0"
BUILD_FLAGS=-ldflags=-s

UNIT_TEST_TAGS="unit"
INTEGRATION_TEST_TAGS="integration"

UNIT_TEST_ONLY_PKGS=$(shell go list -tags ${UNIT_TEST_TAGS} ./... | grep -v "/vendor/" | grep -v "/contrib/" | grep -v "/testing/integration" )
INTEGRATION_TEST_ONLY_PKGS=$(shell go list -tags ${INTEGRATION_TEST_TAGS} ./testing/integration/...)
CHECK_PKGS=$(shell go list ./... | grep -vE '^github.com/trivago/gollum/vendor/')

all: clean vendor test freebsd linux docker mac pi win examples current

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

vendor:
	@go get -u github.com/Masterminds/glide
	@glide cc
	@glide update

test: unit integration

unit:
	@echo "go tests SDK"
	@$(BUILD_ENV) go test $(BUILD_FLAGS) -v -cover -timeout 10s -race -tags ${UNIT_TEST_TAGS} $(UNIT_TEST_ONLY_PKGS)

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
	@$(BUILD_ENV) go test $(BUILD_FLAGS) -v -tags="integration" $(INTEGRATION_TEST_ONLY_PKGS)

pre-commit: vet lint fmt

vet:
	@echo "Running go vet"
	@go vet $(CHECK_PKGS)

lint:
	@echo "Running golint"
	@golint -set_exit_status $(CHECK_PKGS)

fmt:
	@echo "Running go fmt"
	@go fmt $(CHECK_PKGS)

clean:
	@rm -f ./gollum
	@rm -f ./dist/gollum_*.zip
	@go clean

docker: linux
	@echo "Building docker image"
	@docker build --squash -t trivago/gollum:$(VERSION)-latest .

docker-dev:
	@echo "Building development docker image"
	@docker build -t trivago/gollum:$(VERSION)-dev -f Dockerfile-dev .
