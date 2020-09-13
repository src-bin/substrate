# Normal releases are monthly.
VERSION := $(shell date +%Y.%m)

# Emergency releases are daily.
#VERSION := $(shell date +%Y.%m.%d)

# All release tarballs are annotated with a short commit SHA and a dirty bit for the work tree.
COMMIT := $(shell git show --format=%h --no-patch)$(shell git diff --quiet || echo \-dirty)

all:
	go generate ./...

clean:

install:
	ls -1 cmd | xargs -n1 basename | xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(GOBIN)/___ ./cmd/___
	grep -Flr lambda.Start cmd | xargs dirname | xargs -n1 basename | GOARCH=amd64 GOOS=linux xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(GOBIN)/___ ./cmd/___

release:
	mkdir substrate-$(VERSION)-$(COMMIT)-linux
	make install GOARCH=amd64 GOBIN=$(PWD)/substrate-$(VERSION)-$(COMMIT)-linux GOOS=linux
	tar czf substrate-$(VERSION)-$(COMMIT)-linux.tar.gz substrate-$(VERSION)-$(COMMIT)-linux
	rm -f -r substrate-$(VERSION)-$(COMMIT)-linux
	mkdir substrate-$(VERSION)-$(COMMIT)-macos
	make install GOARCH=amd64 GOBIN=$(PWD)/substrate-$(VERSION)-$(COMMIT)-macos GOOS=darwin
	tar czf substrate-$(VERSION)-$(COMMIT)-macos.tar.gz substrate-$(VERSION)-$(COMMIT)-macos
	rm -f -r substrate-$(VERSION)-$(COMMIT)-macos

release-filenames: # for src-bin.co to grab on
	@echo substrate-$(VERSION)-$(COMMIT)-linux.tar.gz
	@echo substrate-$(VERSION)-$(COMMIT)-macos.tar.gz

test:
	go test -race -v ./...

uninstall:
	ls -1 cmd | xargs -I___ rm -f $(GOBIN)/___

.PHONY: all clean install release release-filenames test uninstall
