.PHONY: config

all: build

debug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

fmt:
	go tool golangci-lint fmt

check:
	go tool golangci-lint run --fix --timeout 3m ./...
	go vet ./...

update: bump_go_deps generate

bump_go_deps:
	go get -u ./...
	go mod tidy

generate:
	go generate ./...

race:
	GORACE="race.txt" DEBUG=1 go run -race .

test:
	go test ./...

tail:
	tail -f ~/.config/tf-tui/tf-tui.log

snapshot:
	goreleaser release --snapshot --clean

demo:
	go tool vhs docs/demo.vhs

build:
	go build -o tf-tui cmd/tf-tui/*

run: build
	./tf-tui
