.PHONY: config
debug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

fmt:
	go tool golangci-lint fmt

check:
	go tool golangci-lint run --fix --timeout 3m ./...

update:
	go get -u
	go mod tidy
	make generate

generate:
	go generate

race:
	GORACE="race.txt" DEBUG=1 go run -race .

test:
	go test ./...

tail:
	tail -f ~/.config/tf-tui/tf-tui.log

config:
	vim ~/.config/tf-tui/tf-tui.yaml

snapshot:
	goreleaser release --snapshot --clean

demo:
	go tool vhs docs/demo.vhs
