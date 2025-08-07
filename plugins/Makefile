.PHONY: example

all: generate example

generate:
	protoc --go-plugin_out=. --go-plugin_opt=paths=source_relative plugins.proto

example:
	make -C example
