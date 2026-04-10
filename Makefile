# Build parameters:
#   EDITION — Go build tag for module selection (e.g., edition_lite)
#   APPS    — Comma-separated frontend modules (e.g., system,ai)
EDITION ?=
APPS    ?=

GO_TAGS := $(if $(EDITION),-tags $(EDITION),)

web-build:
ifdef APPS
	APPS=$(APPS) ./scripts/gen-registry.sh
endif
	cd ./web && bun run build
ifdef APPS
	./scripts/gen-registry.sh
endif

web-dev:
	cd ./web && \
	bun run dev

refer-clone:
	cd ./support-files/refer

dev:
	go run -tags dev ./cmd/server

build: web-build
	CGO_ENABLED=0 go build $(GO_TAGS) -o metis ./cmd/server

release: web-build
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(GO_TAGS) -o dist/metis-linux-amd64   ./cmd/server
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(GO_TAGS) -o dist/metis-linux-arm64   ./cmd/server
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(GO_TAGS) -o dist/metis-darwin-amd64  ./cmd/server
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(GO_TAGS) -o dist/metis-darwin-arm64  ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_TAGS) -o dist/metis-windows-amd64.exe ./cmd/server
	@ls -lh dist/

release-license:
	EDITION=edition_license APPS=system,license $(MAKE) web-build
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -tags edition_license -o dist/metis-license-linux-amd64       ./cmd/server
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -tags edition_license -o dist/metis-license-linux-arm64       ./cmd/server
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -tags edition_license -o dist/metis-license-darwin-amd64      ./cmd/server
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -tags edition_license -o dist/metis-license-darwin-arm64      ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags edition_license -o dist/metis-license-windows-amd64.exe ./cmd/server
	@ls -lh dist/metis-license-*

build-license:
	APPS=system,license $(MAKE) web-build
	CGO_ENABLED=0 go build -tags edition_license -o metis-license ./cmd/server

run: build
	./metis

push:
	git add .
	git commit -m "Update"
	git push

init-dev-user:
	go run -tags dev ./cmd/server create-admin --username=admin --password=admin123

.PHONY: web-build web-dev refer-clone dev build release release-license build-license run push init-dev-user
