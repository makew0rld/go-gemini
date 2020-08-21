package gemini

import (
	"fmt"
	"io/ioutil"
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
	// Meta longer than 1024 chars
	_, err := getHeader(strings.NewReader("20 " + strings.Repeat("a", 1025) + "\r\n"))
	if err == nil {
		t.Fatalf("expected to get an error for meta longer than 1024")
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
