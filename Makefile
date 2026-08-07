build:
	go build -trimpath -o bin/signal-server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/ coverage.out coverage.html

.PHONY: build run test vet fmt clean
