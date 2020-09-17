RELEASES= _bin/ciigo-linux-amd64 \
	_bin/ciigo-darwin-amd64

.PHONY: all lint install serve build build-release

all: install

lint:
	golangci-lint run --enable-all \
		--disable=wsl --disable=gomnd --disable=funlen ./...

install:
	go run ./internal/cmd/generate
	go install ./cmd/ciigo-example
	go install ./cmd/ciigo

serve:
	find _example -name "*.html" -delete
	rm -f ./cmd/ciigo-example/static.go
	go run ./internal/cmd/generate
	DEBUG=1 go run ./cmd/ciigo-example

test-parser:
	asciidoctor --attribute stylesheet=empty.css testdata/test.adoc
	go test -v -run=Open .

build-release: _bin $(RELEASES)

_bin:
	mkdir -p _bin

_bin/ciigo-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -o $@ ./cmd/ciigo

_bin/ciigo-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		go build -o $@ ./cmd/ciigo
