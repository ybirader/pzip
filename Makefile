test:
	go test ./...

test-short:
	go test -short ./...

build:
	go build -o ./cmd/pzip ./cmd/pzip && go build -o ./cmd/punzip ./cmd/punzip
