all:
	go generate ./...

clean:

install:
	go install ./cmd/...
	grep -Flr lambda.Start ./cmd | xargs dirname | xargs -n1 basename | GOARCH=amd64 GOOS=linux xargs -I___ go build -o $(GOBIN)/___ ./cmd/___

test:
	go test -race -v ./...

uninstall:
	ls -1 cmd | xargs -I___ rm -f $(GOBIN)/___

.PHONY: all clean install test uninstall
