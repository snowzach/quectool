EXECUTABLE := quectool
GITVERSION := $(shell git describe --dirty --always --tags --long)
LDFLAGS    := -X github.com/snowzach/golib/version.Executable=$(EXECUTABLE) \
              -X github.com/snowzach/golib/version.GitVersion=$(GITVERSION)

.PHONY: default
default: $(EXECUTABLE)

# Host build for local development.
.PHONY: $(EXECUTABLE)
$(EXECUTABLE):
	mkdir -p build
	go build -ldflags "$(LDFLAGS)" -o build/$(EXECUTABLE)

# Cross-compile for the modem (Quectel RM5xx OpenLinux is 32-bit ARMv7).
.PHONY: armv7
armv7: frontend
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
		go build -trimpath -ldflags "-w -s $(LDFLAGS)" -o build/$(EXECUTABLE)-armv7

# Tagged release artifact: armv7 binary + dist/ files in a single tarball
# with sha256. Same artifact CI publishes; usable locally with deploy.sh.
.PHONY: release-tar
release-tar: armv7
	rm -rf build/pkg
	mkdir -p build/pkg
	cp build/$(EXECUTABLE)-armv7  build/pkg/quectool
	cp dist/quectool.yaml         build/pkg/
	cp dist/quectool.service      build/pkg/
	cp dist/install.sh            build/pkg/
	cp dist/uninstall.sh          build/pkg/
	cp dist/README                build/pkg/
	chmod +x build/pkg/install.sh build/pkg/uninstall.sh
	tar -C build/pkg -czf build/$(EXECUTABLE)-$(GITVERSION)-armv7.tar.gz .
	cd build && sha256sum $(EXECUTABLE)-$(GITVERSION)-armv7.tar.gz \
		> $(EXECUTABLE)-$(GITVERSION)-armv7.tar.gz.sha256
	@echo "==> build/$(EXECUTABLE)-$(GITVERSION)-armv7.tar.gz"

# Frontend bundle, embedded into the binary at compile time.
.PHONY: frontend
frontend: types
	cd frontend && [ -d node_modules ] || npm install
	cd frontend && rm -rf dist && npm run build

# Regenerate frontend/src/types/ from the Go structs.
.PHONY: types
types:
	mkdir -p frontend/src/types
	go run github.com/gzuidhof/tygo@latest generate

.PHONY: test
test:
	go test -cover ./...

# Local development:
#   make dev-backend   in one terminal (override MODEM_PORT if not /dev/ttyUSB3)
#   make dev-frontend  in another (vite dev server, proxies /api → :8082)
# Override the production defaults (HTTPS on :443, embedded SPA, /usrdata
# paths) for local laptop dev: plain HTTP on :8080, frontend served from
# the on-disk build, modem on the typical Quectel USB AT port.
.PHONY: dev-backend
dev-backend:
	SERVER_EMBEDDED=false SERVER_TLS=false SERVER_PORT=8080 \
		SERVER_SSH_HOST_KEY_FILE=./host_key \
		SERVER_AUTH_CREDENTIALS_FILE=./credentials \
		MODEM_PORT=/dev/ttyUSB3 \
		go run main.go server

.PHONY: dev-frontend
dev-frontend: types
	cd frontend && [ -d node_modules ] || npm install
	cd frontend && npm run dev

.PHONY: clean
clean:
	rm -rf build frontend/dist embed/public_html frontend/src/types
