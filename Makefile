.PHONY: fmt vet lint test

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./...
