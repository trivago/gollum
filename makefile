.PHONY: all clean freebsd linux mac pi win current restore test
clean:
	@rm -f ./gollum
	@rm -f ./dist/gollum_*.zip

linux:
	@echo "Building for Linux"
	@GOOS=linux GOARCH=amd64 go build -o gollum
	@zip dist/gollum_linux.zip gollum config/*.conf

mac:
	@echo "Building for MacOS X"
	@GOOS=darwin GOARCH=amd64 go build -o gollum
	@zip dist/gollum_mac.zip gollum config/*.conf

freebsd:
	@echo "Building for FreeBSD"
	@GOOS=freebsd GOARCH=amd64 go build -o gollum
	@zip dist/gollum_freebsd.zip gollum config/*.conf

win:
	@echo "Building for Windows"
	@GOOS=windows GOARCH=amd64 go build -o gollum
	@zip dist/gollum_win.zip gollum config/*.conf

pi:
	@echo "Building for Raspberry Pi"
	@GOOS=linux GOARCH=arm go build -o gollum
	@zip dist/gollum_pi.zip gollum config/*.conf

current:
	@go build

restore:
	@go get github.com/tools/godep
	@godep restore

test:
	@go test -cover -v -timeout 10s ./...

all: clean freebsd linux mac pi win current

.DEFAULT_GOAL := current
