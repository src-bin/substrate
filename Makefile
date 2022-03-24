# Normal releases are monthly.
VERSION := $(shell date +%Y.%m)

# Emergency releases are daily.
#VERSION := $(shell date +%Y.%m.%d)

# All release tarballs are annotated with a short commit SHA and a dirty bit for the work tree.
COMMIT := $(shell git show --format=%h --no-patch)$(shell git diff --quiet || echo \-dirty)

ifndef CODEBUILD_BUILD_ID
ENDPOINT := https://src-bin.org/telemetry/
else
ENDPOINT := https://src-bin.com/telemetry/
endif

all:
	go generate ./lambdautil # dependency of several packages with go:generate directives
	go generate ./cmd/substrate-intranet # dependency of cmd/substrate-create-admin-account's go:generate directives
	go generate ./... # the rest of the go:generate directives

clean:
	find -name dispatch-map.go -delete
	find -name \*.html.go -delete
	find -name \*.template.go -delete
	find terraform -name \*-global.go -o -name \*-regional.go -delete
	rm -f cmd/substrate-create-admin-account/substrate-intranet*
	rm -f -r substrate-*-*-*
	rm -f substrate-*-*-*.tar.gz

install:
	find ./cmd -maxdepth 1 -mindepth 1 -not -name substrate-intranet -type d | xargs -n1 basename | xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/telemetry.Endpoint=$(ENDPOINT) -X github.com/src-bin/substrate/terraform.TerraformVersion=$(shell cat terraform-version.txt) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(shell go env GOBIN)/___ ./cmd/___
	find ./cmd/substrate -maxdepth 1 -mindepth 1 -type d -printf $(shell go env GOBIN)/substrate-%P\\n | xargs -n1 ln -f -s substrate

release:
ifndef CODEBUILD_BUILD_ID
	@echo you probably meant to \`make -C release\` in src-bin/, not \`make release\` in substrate/
	@false
endif
	GOARCH=amd64 GOOS=linux make tarball
	GOARCH=arm64 GOOS=linux make tarball
	GOARCH=amd64 GOOS=darwin make tarball
	GOARCH=arm64 GOOS=darwin make tarball

release-filenames: # for src-bin.com to grab on
	@echo substrate-$(VERSION)-$(COMMIT)-linux-amd64.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-linux-arm64.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-darwin-amd64.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-darwin-arm64.tar.gz

release-version: # for src-bin.com to grab on
	@echo $(VERSION)

tarball:
	rm -f -r substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH) # makes debugging easier
	mkdir substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	GOBIN=$(PWD)/substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH) make install
	tar czf substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH).tar.gz substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	rm -f -r substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)

test:
	go test -race -v ./...

uninstall:
	find ./cmd -maxdepth 1 -mindepth 1 -type d -printf $(shell go env GOBIN)/%P\\n | xargs rm -f
	find ./cmd/substrate -maxdepth 1 -mindepth 1 -type d -printf $(shell go env GOBIN)/substrate-%P\\n | xargs rm -f

.PHONY: all clean install release release-filenames tarball test uninstall
