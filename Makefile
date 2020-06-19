all:
	go generate ./...

clean:

install:
	go install ./cmd/...
	grep -Flr lambda.Start ./cmd | xargs dirname | xargs -n1 basename | GOARCH=amd64 GOOS=linux xargs -I_ go build -o $(GOBIN)/_ ./cmd/_

test:
	go test -race -v ./...

.PHONY: all clean install test
