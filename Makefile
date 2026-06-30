.PHONY: all build wails-build wails-dev test coverage swag run clean

all: build

build:
	go build -o bin/aibrow ./cmd/aibrow

wails-build:
	wails build -clean

wails-dev:
	wails dev

test:
	go test -race ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

swag:
	go run github.com/swaggo/swag/cmd/swag init -g cmd/aibrow/main.go -o docs

run: build
	./bin/aibrow

clean:
	rm -rf bin build/bin coverage.out
