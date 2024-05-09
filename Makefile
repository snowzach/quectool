EXECUTABLE := quectool
GITVERSION := $(shell git describe --dirty --always --tags --long)
GOPATH ?= ${HOME}/go
PACKAGENAME := $(shell go list -m -f '{{.Path}}')

.PHONY: default
default: ${EXECUTABLE}
	
.PHONY: ${EXECUTABLE}
${EXECUTABLE}:
	# Compiling...
	mkdir -p build
	go build -ldflags "-X github.com/snowzach/golib/version.Executable=${EXECUTABLE} -X github.com/snowzach/golib/version.GitVersion=${GITVERSION}" -o build/${EXECUTABLE}

.PHONY: test
test: tools mocks
	go test -cover ./...

.PHONY: lint
lint:
	docker run --rm -v ${PWD}:/app -w /app golangci/golangci-lint:latest golangci-lint run -v --timeout 5m

.PHONY: armv7
armv7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-w -s -X github.com/snowzach/golib/version.Executable=${EXECUTABLE} -X github.com/snowzach/golib/version.GitVersion=${GITVERSION}" -o build/${EXECUTABLE}-armv7

.PHONY: windows
windows:
	CGO_ENABLED=0 GOOS=windows go build -ldflags "-w -s -X github.com/snowzach/golib/version.Executable=${EXECUTABLE} -X github.com/snowzach/golib/version.GitVersion=${GITVERSION}" -o build/${EXECUTABLE}-windows.exe

.PHONY: atcmd-armv7
atcmd-armv7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-w -s -X github.com/snowzach/golib/version.Executable=${EXECUTABLE} -X github.com/snowzach/golib/version.GitVersion=${GITVERSION}" -o build/atcmd-armv7 cmd/atcmd/atcmd.go

.PHONY: atcmd-windows
atcmd-windows:
	CGO_ENABLED=0 GOOS=windows go build -ldflags "-w -s -X github.com/snowzach/golib/version.Executable=${EXECUTABLE} -X github.com/snowzach/golib/version.GitVersion=${GITVERSION}" -o build/atcmd-windows.exe cmd/atcmd/atcmd.go


