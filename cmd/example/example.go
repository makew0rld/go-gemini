package main

import (
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	gemini "git.sr.ht/~yotam/go-gemini"
)

type ExampleHandler struct {
}

func (h ExampleHandler) Handle(r gemini.Request) gemini.Response {
	u, err := url.Parse(r.URL)
	if err != nil {
		body := ioutil.NopCloser(strings.NewReader(err.Error()))
		return gemini.Response{40, "text/gemini", body}
	}

	if u.Path != "/" {
		body := ioutil.NopCloser(strings.NewReader("Not Found"))
		return gemini.Response{50, "text/gemini", body}
	}

	body := ioutil.NopCloser(strings.NewReader("Hello World"))
	return gemini.Response{20, "text/gemini", body}
}

func main() {
	handler := ExampleHandler{}

	err := gemini.ListenAndServe(":1965", "server.crt", "server.key", handler)
	if err != nil {
		log.Fatal(err)
	}
}
