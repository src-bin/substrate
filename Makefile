#VERSION := $(shell date +%Y.%m) # once we get into a release cadence, this is how versions will be formatted
VERSION := $(shell date +%Y.%m.%d.%H.%M) # but things are moving fast right now and versions should express that

all:
	go generate ./...

clean:

install:
	go install -ldflags "-X github.com/src-bin/substrate/version.Version=$(VERSION)" ./cmd/...
	grep -Flr lambda.Start ./cmd | xargs dirname | xargs -n1 basename | GOARCH=amd64 GOOS=linux xargs -I___ go build -ldflags "-X github.com/src-bin/substrate/version.Version=$(VERSION)" -o $(GOBIN)/___ ./cmd/___

test:
	go test -race -v ./...

uninstall:
	ls -1 cmd | xargs -I___ rm -f $(GOBIN)/___

.PHONY: all clean install test uninstall
