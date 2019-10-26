all: build

build: gemini-example

gemini-example: cmd/example/*.go *.go
	go build -o gemini-example git.sr.ht/~yotam/go-gemini/cmd/example

clean:
	rm -rf gemini-example

.PHONY: all build clean
