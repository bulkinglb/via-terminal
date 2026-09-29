VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build test release publish clean

build:
	go build -ldflags "$(LDFLAGS)"

test:
	go vet ./...
	go test ./...

# Binaries for every platform in dist/: tar.gz for Linux and macOS so the
# executable bit survives, a plain .exe for Windows, plus checksums.
release: test
	rm -rf dist
	mkdir dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=via-terminal-$(VERSION)-$$os-$$arch; \
		echo "building $$name"; \
		if [ $$os = windows ]; then \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$$name.exe . || exit 1; \
		else \
			mkdir dist/$$name && cp README.md LICENSE dist/$$name/ && \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$$name/via-terminal . && \
			tar -C dist -czf dist/$$name.tar.gz $$name && rm -r dist/$$name || exit 1; \
		fi; \
	done
	cd dist && sha256sum * > checksums.txt

# make publish VERSION=v0.1.0 tags the current commit and puts the release
# on GitHub. It needs a clean working tree so the binaries match the tag.
publish:
	@case "$(VERSION)" in v[0-9]*) ;; *) echo "set a version: make publish VERSION=v0.1.0"; exit 1;; esac
	@git diff --quiet HEAD || { echo "commit your changes first"; exit 1; }
	$(MAKE) release VERSION=$(VERSION)
	git tag $(VERSION)
	git push origin $(VERSION)
	gh release create $(VERSION) dist/* --verify-tag --title $(VERSION) --generate-notes

clean:
	rm -rf dist via-terminal
