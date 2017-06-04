.PHONY: all clean docker docker-dev install freebsd linux mac pi win examples current vendor test example
.DEFAULT_GOAL := current

VERSION=0.4.6.1
BUILD_FLAGS=GO15VENDOREXPERIMENT=1 GORACE="halt_on_error=0"

all: clean vendor test freebsd linux docker mac pi win examples current

freebsd:
	@echo "Building for FreeBSD/x64"
	@GOOS=freebsd GOARCH=amd64 $(BUILD_FLAGS) go build -o gollum
	@rm -f dist/gollum-$(VERSION)-FreeBSD_x64.zip
	@zip dist/gollum-$(VERSION)-FreeBSD_x64.zip gollum
    
linux:
	@echo "Building for Linux/x64"
	@GOOS=linux GOARCH=amd64 $(BUILD_FLAGS) go build -o gollum
	@rm -f dist/gollum-$(VERSION)-Linux_x64.zip
	@zip dist/gollum-$(VERSION)-Linux_x64.zip gollum

mac:
	@echo "Building for MacOS X (MacOS/x64)"
	@GOOS=darwin GOARCH=amd64 $(BUILD_FLAGS) go build -o gollum
	@rm -f dist/gollum-$(VERSION)-MacOS_x64.zip
	@zip dist/gollum-$(VERSION)-MacOS_x64.zip gollum

pi:
	@echo "Building for Raspberry Pi (Linux/ARMv6)"
	@GOOS=linux GOARCH=arm GOARM=6 $(BUILD_FLAGS) go build -o gollum
	@rm -f dist/gollum-$(VERSION)-Linux_Arm6.zip
	@zip dist/gollum-$(VERSION)-Linux_Arm6.zip gollum

win:
	@echo "Building for Windows/x64"
	@GOOS=windows GOARCH=amd64 $(BUILD_FLAGS) go build -o gollum.exe
	@rm -f dist/gollum-$(VERSION)-Windows_x64.zip
	@zip dist/gollum-$(VERSION)-Windows_x64.zip gollum

current:
	@$(BUILD_FLAGS) go build

install: current
	@go install
    
examples:
	@echo "Building Examples"
	@zip -j dist/gollum-$(VERSION)-Examples.zip config/*.conf

vendor:
	@go get -u github.com/Masterminds/glide
	@glide cc
	@glide update

test:
	@$(BUILD_FLAGS) go test -cover -v -timeout 10s -race $$(go list ./...|grep -v vendor)

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