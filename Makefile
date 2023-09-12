test:
	go test ./...

test-short:
	go test -short ./...

lint:
	golangci-lint run --enable-all
