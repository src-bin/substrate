# If the commit's tagged, that's the version. If it's not, use a short commit
# SHA. In either case, annotate builds from dirty work trees.
VERSION ?= $(shell git describe --exact-match --tags HEAD 2>/dev/null || git show --format=%h --no-patch)$(shell git diff --quiet || echo \-dirty)

ifndef CODEBUILD_BUILD_ID
ENDPOINT := https://src-bin.org/telemetry/
else
ENDPOINT := https://src-bin.com/telemetry/
endif

all: go-generate
	# TODO go vet ...

clean:
	find . -name dispatch-map.go -delete
	find . -name dispatch-map-\*.go -delete
	find . -name \*.html.go -delete
	find . -name \*.js.go -delete
	find . -name \*.template.go -delete
	find . -name \*.tf.go -delete
	find terraform -name \*-global.go -o -name \*-regional.go -delete
	rm -f cmd/substrate/intranet-zip/bootstrap
	rm -f cmd/substrate/intranet-zip/substrate-intranet.zip
	rm -f -r substrate-*-*-*
	rm -f substrate-*-*-*.tar.gz
	rm -f substrate.version
	rm -f terraform/peering-connection.go
	rm -f -r upgrade

deps:
	go get -u ./...

go-generate:
	go generate -x ./lambdautil # dependency of several packages with go:generate directives
	go generate -x ./cmd/substrate-intranet     # dependencies of...
	go generate -x ./cmd/substrate-intranet/... # ...cmd/substrate/intranet-zip's...
	go generate -x ./terraform                  # ...go:generate directives
	go generate -x ./... # the rest of the go:generate directives

go-generate-intranet:
	env GOARCH=arm64 GOOS=linux go build -ldflags "-X github.com/src-bin/substrate/telemetry.Endpoint=$(ENDPOINT) -X github.com/src-bin/substrate/terraform.DefaultRequiredVersion=$(shell cat terraform.version) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o cmd/substrate/intranet-zip/bootstrap ./cmd/substrate-intranet

install:
	find ./cmd -maxdepth 1 -mindepth 1 -not -name substrate-intranet -type d | xargs -n1 basename | xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/telemetry.Endpoint=$(ENDPOINT) -X github.com/src-bin/substrate/terraform.DefaultRequiredVersion=$(shell cat terraform.version) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(shell go env GOBIN)/___ ./cmd/___

release: release-darwin release-linux

release-darwin:
ifdef CODEBUILD_BUILD_ID
	aws s3 ls s3://$(S3_BUCKET)/substrate/substrate-$(VERSION)-darwin-amd64.tar.gz
	aws s3 ls s3://$(S3_BUCKET)/substrate/substrate-$(VERSION)-darwin-arm64.tar.gz
else
	make tarball GOARCH=amd64 GOOS=darwin VERSION=$(VERSION)
	make tarball GOARCH=arm64 GOOS=darwin VERSION=$(VERSION)
endif

release-linux:
	make tarball GOARCH=amd64 GOOS=linux VERSION=$(VERSION)
	make tarball GOARCH=arm64 GOOS=linux VERSION=$(VERSION)
ifndef CODEBUILD_BUILD_ID
	@echo you probably meant to \`make -C release\` in src-bin/, not \`make release\` in substrate/
endif

tarball:
	rm -f -r substrate-$(VERSION)-$(GOOS)-$(GOARCH) # makes debugging easier
	mkdir substrate-$(VERSION)-$(GOOS)-$(GOARCH)
	mkdir substrate-$(VERSION)-$(GOOS)-$(GOARCH)/bin
	mkdir substrate-$(VERSION)-$(GOOS)-$(GOARCH)/opt substrate-$(VERSION)-$(GOOS)-$(GOARCH)/opt/bin
	mkdir substrate-$(VERSION)-$(GOOS)-$(GOARCH)/src
	GOBIN=$(PWD)/substrate-$(VERSION)-$(GOOS)-$(GOARCH)/opt/bin make install
	mv substrate-$(VERSION)-$(GOOS)-$(GOARCH)/opt/bin/substrate substrate-$(VERSION)-$(GOOS)-$(GOARCH)/bin
	git archive HEAD | tar -C substrate-$(VERSION)-$(GOOS)-$(GOARCH)/src -x
	tar -c -f substrate-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -z substrate-$(VERSION)-$(GOOS)-$(GOARCH)
	rm -f -r substrate-$(VERSION)-$(GOOS)-$(GOARCH)

test:
	go clean -testcache
	go test -race -timeout 0 -v ./...

uninstall:
	find ./cmd -maxdepth 1 -mindepth 1 -type d -printf $(shell go env GOBIN)/%P\\n | xargs rm -f

.PHONY: all clean deps install release release-darwin release-linux tarball test uninstall
