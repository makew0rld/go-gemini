package main

import (
	"io/ioutil"
	"log"
	"strings"

	gemini "github.com/makeworld-the-better-one/go-gemini"
)

type ExampleHandler struct {
}

func (h ExampleHandler) Handle(r gemini.Request) *gemini.Response {
	if r.URL.Path != "/" {
		body := ioutil.NopCloser(strings.NewReader("Not Found"))
		return &gemini.Response{50, "text/gemini", body, nil}
	}

	body := ioutil.NopCloser(strings.NewReader("Hello World"))
	return &gemini.Response{20, "text/gemini", body, nil}
}

func main() {
	handler := ExampleHandler{}

	err := gemini.ListenAndServe("", "server.crt", "server.key", handler)
	if err != nil {
		log.Fatal(err)
	}
}
