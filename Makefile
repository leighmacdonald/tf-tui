debug:
	dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient

fmt:
	go tool gci write . --skip-generated -s standard -s default
	go tool gofumpt -l -w .

check: lint_golangci

lint_golangci:
	go tool golangci-lint run --fix --timeout 3m ./...

update:
	go get -u
	go mod tidy
