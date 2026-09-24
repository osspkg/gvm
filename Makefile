SHELL=/bin/bash

.PHONY: install
install:
	PATH="$$(go env GOROOT)/bin:$$PATH" go install go.osspkg.com/goppy/v3/cmd/goppy@latest
	PATH="$$(go env GOROOT)/bin:$$PATH" goppy setup-lib

.PHONY: lint
lint:
	PATH="$$(go env GOROOT)/bin:$$PATH" goppy lint

.PHONY: license
license:
	PATH="$$(go env GOROOT)/bin:$$PATH" goppy license

.PHONY: build
build:
	PATH="$$(go env GOROOT)/bin:$$PATH" goppy build --arch=amd64

.PHONY: tests
tests:
	PATH="$$(go env GOROOT)/bin:$$PATH" goppy test

.PHONY: pre-commit
pre-commit: install license lint tests build

.PHONY: ci
ci: pre-commit
