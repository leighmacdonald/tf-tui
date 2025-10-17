.PHONY: config

all: build

debug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

fmt:
	golangci-lint fmt

check:
	golangci-lint run --fix --disable unused --timeout 3m ./...
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
	vhs docs/demo.vhs

build:
	go build -o tf-tui internal/cmd/tf-tui/*

run: build
	./tf-tui
