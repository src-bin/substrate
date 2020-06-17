all:

clean:

install:
	go generate ./...
	go install ./cmd/...
	grep -Flr lambda.Start ./cmd | xargs dirname | GOARCH=amd64 GOOS=linux xargs -I_ go build -o $(GOBIN)/bin/_ _

test:

.PHONY: all clean install test
