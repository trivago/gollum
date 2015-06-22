.PHONY: all clean freebsd linux mac pi win
clean:
	@rm -f ./gollum
	@rm -f ./gollum_*.tar.gz

linux:
	@echo "Building for Linux"
	@GOOS=linux GOARCH=amd64 go build -o gollum
	@tar czf gollum_linux.tar.gz gollum config

mac:
	@echo "Building for MacOS X"
	@GOOS=darwin GOARCH=amd64 go build -o gollum
	@tar czf gollum_mac.tar.gz gollum config

freebsd:
	@echo "Building for FreeBSD"
	@GOOS=freebsd GOARCH=amd64 go build -o gollum
	@tar czf gollum_freebsd.tar.gz gollum config

win:
	@echo "Building for Windows"
	@GOOS=windows GOARCH=amd64 go build -o gollum
	@tar czf gollum_win.tar.gz gollum config

pi:
	@echo "Building for Raspberry Pi"
	@GOOS=linux GOARCH=arm go build -o gollum
	@tar czf gollum_pi.tar.gz gollum config

current:
	@go build

restore:
	@go get github.com/tools/godep
	@godep restore

test:
	@go test -v ./...

all: clean freebsd linux mac pi win

.DEFAULT_GOAL := current
