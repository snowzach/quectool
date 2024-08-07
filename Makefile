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





.PHONY: assets
assets: bindata/static/js/gotty.js.map \
	bindata/static/js/gotty.js \
	bindata/static/index.html \
	bindata/static/icon.svg \
	bindata/static/favicon.ico \
	bindata/static/css/index.css \
	bindata/static/css/xterm.css \
	bindata/static/css/xterm_customize.css \
	bindata/static/manifest.json \
	bindata/static/icon_192.png

all: gotty

bindata/static bindata/static/css bindata/static/js:
	mkdir -p $@

bindata/static/%: resources/% | bindata/static/css 
	cp "$<" "$@"

bindata/static/css/%.css: resources/%.css | bindata/static 
	cp "$<" "$@"

bindata/static/css/xterm.css: js/node_modules/xterm/css/xterm.css | bindata/static
	cp "$<" "$@"

js/node_modules/xterm/dist/xterm.css:
	cd js && \
	npm install

bindata/static/js/gotty.js.map bindata/static/js/gotty.js: js/src/* | js/node_modules/webpack
	cd js && \
	npx webpack --mode=$(WEBPACK_MODE)

js/node_modules/webpack:
	cd js && \
	npm install

