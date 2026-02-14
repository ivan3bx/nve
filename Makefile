SHELL := /bin/bash

help: _help_

_help_:
	@echo make build         - build and push release
	@echo make test          - run all tests
	@echo make build-local   - build locally
	@echo make release-local - build and archive for release

.PHONY: clean
clean:
	rm -rf ./dist

.PHONY: build
build: .goreleaser.yml
	goreleaser build --clean --snapshot -f .goreleaser.yml

.PHONY: build-local
build-local:
	goreleaser build --clean --single-target --snapshot -f .goreleaser.yml

.PHONY: test
test:
	go test ./... --tags=fts5 --count=1

.PHONY: test-tui
test-tui:
	go test --tags="fts5 integration" -run TestTUI --count=1 -v

.PHONY: release-local
release-local:
	goreleaser release --snapshot --rm-dist -f .goreleaser.yml
