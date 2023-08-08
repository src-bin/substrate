# If the commit's tagged, that's the version. If it's not, use an amusing,
# dotted, second-resolution timestamp as the version.
VERSION ?= $(shell git describe --exact-match --tags HEAD 2>/dev/null || date +%Y.%m.%d.%H.%M.%S)

# All release tarballs are annotated with a short commit SHA and a dirty bit for the work tree.
COMMIT ?= $(shell git show --format=%h --no-patch)$(shell git diff --quiet || echo \-dirty)

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
	find . -name \*.template.go -delete
	find . -name \*.tf.go -delete
	find terraform -name \*-global.go -o -name \*-regional.go -delete
	rm -f cmd/substrate/intranet-zip/bootstrap
	rm -f cmd/substrate/intranet-zip/substrate-intranet.zip
	rm -f -r substrate-*-*-*
	rm -f substrate-*-*-*.tar.gz
	rm -f terraform/peering-connection.go

deps:
	go get -u ./...
	go get -u golang.org/x/tools/cmd/goimports

go-generate:
	go generate ./lambdautil # dependency of several packages with go:generate directives
	go generate ./cmd/substrate-intranet     # dependency of cmd/substrate/intranet-zip's...
	go generate ./cmd/substrate-intranet/... # ...go:generate directives
	go generate ./... # the rest of the go:generate directives

go-generate-intranet:
	env GOARCH=arm64 GOOS=linux go build -ldflags "-X github.com/src-bin/substrate/telemetry.Endpoint=$(ENDPOINT) -X github.com/src-bin/substrate/terraform.TerraformVersion=$(shell cat terraform.version) -X github.com/src-bin/substrate/version.Commit=$(COMMIT) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o cmd/substrate/intranet-zip/bootstrap ./cmd/substrate-intranet

install:
	find ./cmd -maxdepth 1 -mindepth 1 -not -name substrate-intranet -type d | xargs -n1 basename | xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/telemetry.Endpoint=$(ENDPOINT) -X github.com/src-bin/substrate/terraform.TerraformVersion=$(shell cat terraform.version) -X github.com/src-bin/substrate/version.Commit=$(COMMIT) -X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(shell go env GOBIN)/___ ./cmd/___

release:
ifndef CODEBUILD_BUILD_ID
	@echo you probably meant to \`make -C release\` in src-bin/, not \`make release\` in substrate/
	@false
endif
	make tarball GOARCH=amd64 GOOS=darwin VERSION=$(VERSION)
	make tarball GOARCH=amd64 GOOS=darwin VERSION=$(VERSION) COMMIT=trial
	make tarball GOARCH=amd64 GOOS=linux VERSION=$(VERSION)
	make tarball GOARCH=amd64 GOOS=linux VERSION=$(VERSION) COMMIT=trial
	make tarball GOARCH=arm64 GOOS=darwin VERSION=$(VERSION)
	make tarball GOARCH=arm64 GOOS=darwin VERSION=$(VERSION) COMMIT=trial
	make tarball GOARCH=arm64 GOOS=linux VERSION=$(VERSION)
	make tarball GOARCH=arm64 GOOS=linux VERSION=$(VERSION) COMMIT=trial
	echo $(VERSION) >substrate.version

tarball:
	rm -f -r substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH) # makes debugging easier
	mkdir substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	mkdir substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/bin
	mkdir substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/opt substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/opt/bin
	mkdir substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/src
	GOBIN=$(PWD)/substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/opt/bin make install
	mv substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/opt/bin/substrate substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/bin
	git archive HEAD | tar -C substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)/src -x
	tar -c -f substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH).tar.gz -z substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)
	rm -f -r substrate-$(VERSION)-$(COMMIT)-$(GOOS)-$(GOARCH)

test:
	go clean -testcache
	go test -race -timeout 0 -v ./...

uninstall:
	find ./cmd -maxdepth 1 -mindepth 1 -type d -printf $(shell go env GOBIN)/%P\\n | xargs rm -f
	find ./cmd/substrate -maxdepth 1 -mindepth 1 -type d -printf $(shell go env GOBIN)/substrate-%P\\n | xargs rm -f

.PHONY: all clean deps install release tarball test uninstall
