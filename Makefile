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
	goreleaser build --rm-dist --snapshot -f .goreleaser.yml

.PHONY: build-local
build-local:
	goreleaser build --single-target --rm-dist --snapshot -f .goreleaser.yml

.PHONY: test
test:
	go test -v ./...

.PHONY: release-local
release-local:
	goreleaser release --snapshot --rm-dist -f .goreleaser.yml