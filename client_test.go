package gemini

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func compareResponses(expected, given *Response) (diff string) {
	diff = cmp.Diff(expected.Meta, given.Meta)
	if diff != "" {
		return
	}

	diff = cmp.Diff(expected.Meta, given.Meta)
	if diff != "" {
		return
	}

	expectedBody, err := ioutil.ReadAll(expected.Body)
	if err != nil {
		return fmt.Sprintf("failed to get expected body: %v", err)
	}

	givenBody, err := ioutil.ReadAll(given.Body)
	if err != nil {
		return fmt.Sprintf("failed to get givenponse body: %v", err)
	}

	diff = cmp.Diff(expectedBody, givenBody)
	return
}

func TestGetResponse(t *testing.T) {
	tests := []struct {
		file     string
		expected Response
	}{
		{"resources/tests/simple_response", Response{20, "text/gemini", ioutil.NopCloser(strings.NewReader("This is the content of the page\r\n")), nil}},
	}

	for _, tc := range tests {
		f, err := os.Open(tc.file)
		if err != nil {
			t.Fatalf("failed to get test case file %s: %v", tc.file, err)
		}

		res := Response{}
		err = getResponse(&res, f)
		if err != nil {
			t.Fatalf("failed to parse response %s: %v", tc.file, err)
		}

		diff := compareResponses(&tc.expected, &res)
		if diff != "" {
			t.Fatalf(diff)
		}
	}

}

func TestGetResponseEmptyResponse(t *testing.T) {
	err := getResponse(&Response{}, ioutil.NopCloser(strings.NewReader("")))
	if err == nil {
		t.Fatalf("expected to get an error for empty response, got nil instead")
	}
}

func TestGetResponseInvalidStatus(t *testing.T) {
	err := getResponse(&Response{}, ioutil.NopCloser(strings.NewReader("AA\tmeta\r\n")))
	if err == nil {
		t.Fatalf("expected to get an error for invalid status response, got nil instead")
	}
}

func TestGetHeaderLongMeta(t *testing.T) {
	// Meta longer than allowed
	_, err := getHeader(strings.NewReader("20 " + strings.Repeat("a", MetaMaxLength+1) + "\r\n"))
	if err == nil {
		t.Fatalf(fmt.Sprintf("expected to get an error for meta longer than %d", MetaMaxLength))
	}
}

func TestGetHeaderOnlyLF(t *testing.T) {
	// Meta longer than 1024 chars
	_, err := getHeader(strings.NewReader("20 test" + "\n"))
	if err == nil {
		t.Fatalf("expected to get an error for header ending only in LF")
	}
}

func TestGetHeaderNoSpace(t *testing.T) {
	_, err := getHeader(strings.NewReader("20\r\n"))
	if err == nil {
		t.Fatalf("expected to get an error for header with no space")
	}
}

func parse(s string) *url.URL {
	p, _ := url.Parse(s)
	return p
}

func TestGetHost(t *testing.T) {
	tests := []struct {
		host string
		url  string
	}{
		{"example.com:1965", "gemini://example.com:1965"},
		{"example.com:1965", "gemini://example.com"},
		{"example.com:1965", "gemini://example.com/test//"},
		{"example.com:123", "gemini://example.com:123"},
		{"example.com:123", "gemini://example.com:123/test//"},
		{"0.0.0.0:1965", "gemini://0.0.0.0:1965"},
		{"0.0.0.0:1965", "gemini://0.0.0.0"},
		{"0.0.0.0:1965", "gemini://0.0.0.0/test//"},
		{"0.0.0.0:123", "gemini://0.0.0.0:123"},
		{"0.0.0.0:123", "gemini://0.0.0.0:123/test//"},
		{"[::1]:1965", "gemini://[::1]:1965"},
		{"[::1]:1965", "gemini://[::1]"},
		{"[::1]:1965", "gemini://[::1]/test//"},
		{"[::1]:123", "gemini://[::1]:123"},
		{"[::1]:123", "gemini://[::1]:123/test//"},
	}

	for _, tc := range tests {
		host := getHost(parse(tc.url))
		if tc.host != host {
			t.Errorf("Got %s but expected %s for URL %s", host, tc.host, tc.url)
		}
	}
}
