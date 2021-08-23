# Normal releases are monthly.
VERSION := $(shell date +%Y.%m)

# Emergency releases are daily.
#VERSION := $(shell date +%Y.%m.%d)

# All release tarballs are annotated with a short commit SHA and a dirty bit for the work tree.
COMMIT := $(shell git show --format=%h --no-patch)$(shell git diff --quiet || echo \-dirty)

all:
	go generate ./...

clean:
	rm -f -r substrate-*-*-*
	rm -f substrate-*-*-*.tar.gz

install:
	find ./cmd -maxdepth 1 -mindepth 1 -not -name substrate-intranet -type d | xargs -n1 basename | xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/terraform.TerraformVersion=$(shell cat terraform-version.txt) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(shell go env GOBIN)/___ ./cmd/___
	echo '#!/bin/sh' >$(shell go env GOBIN)/substrate-apigateway-authenticator # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(shell go env GOBIN)/substrate-apigateway-authorizer # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(shell go env GOBIN)/substrate-apigateway-index # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(shell go env GOBIN)/substrate-credential-factory # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(shell go env GOBIN)/substrate-instance-factory # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(shell go env GOBIN)/substrate-intranet # change to `rm -f` in 2021.09
	chmod +x $(shell go env GOBIN)/substrate-*
	ln -f -s substrate $(shell go env GOBIN)/substrate-whoami

release:
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
	rm -f $(shell go env GOBIN)/substrate-apigateway-authenticator # remove in 2021.09
	rm -f $(shell go env GOBIN)/substrate-apigateway-authorizer # remove in 2021.09
	rm -f $(shell go env GOBIN)/substrate-apigateway-index # remove in 2021.09
	rm -f $(shell go env GOBIN)/substrate-credential-factory # remove in 2021.09
	rm -f $(shell go env GOBIN)/substrate-instance-factory # remove in 2021.09
	rm -f $(shell go env GOBIN)/substrate-whoami

.PHONY: all clean install release release-filenames tarball test uninstall
