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
	ls -1 cmd | xargs -n1 basename | xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/terraform.TerraformVersion=$(shell cat terraform-version.txt) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(GOBIN)/___ ./cmd/___
	grep -Flr lambda.Start cmd | xargs -n1 dirname | xargs -n1 basename | GOARCH=amd64 GOOS=linux xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/terraform.TerraformVersion=$(shell cat terraform-version.txt) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(GOBIN)/___ -tags netgo ./cmd/___
	echo '#!/bin/sh' >$(GOBIN)/substrate-apigateway-authenticator # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(GOBIN)/substrate-apigateway-index # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(GOBIN)/substrate-credential-factory # change to `rm -f` in 2021.09
	echo '#!/bin/sh' >$(GOBIN)/substrate-instance-factory # change to `rm -f` in 2021.09
	chmod +x $(GOBIN)/substrate-*

release:
	make tarball GOARCH=amd64 GOOS=linux
	make tarball GOARCH=arm64 GOOS=linux
	make tarball GOARCH=amd64 GOOS=darwin
	make tarball GOARCH=arm64 GOOS=darwin

release-filenames: # for src-bin.co to grab on
	@echo substrate-$(VERSION)-$(COMMIT)-linux-amd64.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-linux-arm64.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-darwin-amd64.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-darwin-arm64.tar.gz

release-version: # for src-bin.co to grab on
	@echo $(VERSION)

tarball:
	rm -f -r substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH) # makes debugging easier
	mkdir substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	make install GOBIN=$(PWD)/substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	tar czf substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH).tar.gz substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	rm -f -r substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)

test:
	go test -race -v ./...

uninstall:
	ls -1 cmd | xargs -I___ rm -f $(GOBIN)/___

.PHONY: all clean install release release-filenames tarball test uninstall
