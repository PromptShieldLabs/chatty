.PHONY: build test clean install version tag

# Get version from git tags, or use 0.1.0 if no tags exist
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.4")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o chatty ./cmd/chatty

test:
	go test ./...

clean:
	rm -f chatty

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/chatty

run: build
	./chatty

version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

tag:
	@echo "Current version: $(VERSION)"
	@read -p "Enter new version (e.g., v0.2.0): " new_version; \
	git tag -a $$new_version -m "Release $$new_version"; \
	echo "Tagged $$new_version. Push with: git push origin $$new_version"
