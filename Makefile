all:
	go generate ./...

clean:

install:
	go install ./cmd/...
	grep -Flr lambda.Start ./cmd | xargs dirname | GOARCH=amd64 GOOS=linux xargs -I_ go build -o $(GOBIN)/bin/_ _

test:
	go test -race -v ./...

.PHONY: all clean install test
