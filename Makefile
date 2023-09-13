test:
	go test ./...

test-short:
	go test -short ./...

build:
	go build -o ./cmd/cli/pzip ./cmd/cli
