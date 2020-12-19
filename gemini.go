package gemini

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	URLMaxLength  = 1024
	MetaMaxLength = 1024
)

// Gemini status codes as defined in the Gemini spec Appendix 1.
const (
	StatusInput          = 10
	StatusSensitiveInput = 11

	StatusSuccess = 20

	StatusRedirect          = 30
	StatusRedirectTemporary = 30
	StatusRedirectPermanent = 31

	StatusTemporaryFailure = 40
	StatusUnavailable      = 41
	StatusCGIError         = 42
	StatusProxyError       = 43
	StatusSlowDown         = 44

	StatusPermanentFailure    = 50
	StatusNotFound            = 51
	StatusGone                = 52
	StatusProxyRequestRefused = 53
	StatusBadRequest          = 59

	StatusClientCertificateRequired = 60
	StatusCertificateNotAuthorised  = 61
	StatusCertificateNotValid       = 62
)

// All the statuses between 10 and 62 that are invalid
var invalidStatuses = []int{
	12, 13, 14, 15, 16, 17, 18, 19,
	21, 22, 23, 24, 25, 26, 27, 28, 29,
	32, 33, 34, 35, 36, 37, 38, 39,
	45, 46, 47, 48, 49,
	54, 55, 56, 57, 58,
}

// SimplifyStatus simplify the response status by omiting the detailed second digit of the status code.
func SimplifyStatus(status int) int {
	return (status / 10) * 10
}

// IsStatusValid checks whether an int status is covered by the spec.
func IsStatusValid(status int) bool {
	if status < 10 || status > 62 {
		return false
	}
	for _, v := range invalidStatuses {
		if status == v {
			return false
		}
	}
	return true
}

// QueryEscape provides URL query escaping in a way that follows the Gemini spec.
// It is the same as url.PathEscape, but it also replaces the +, because Gemini
// requires percent-escaping for queries.
func QueryEscape(query string) string {
	return strings.ReplaceAll(url.PathEscape(query), "+", "%2B")
}

// QueryUnescape is the same as url.PathUnescape
func QueryUnescape(query string) (string, error) {
	return url.PathUnescape(query)
}

type Error struct {
	Err    error
	Status int
}

func (e Error) Error() string {
	return fmt.Sprintf("Status %d: %v", e.Status, e.Err)
}

func (e Error) Unwrap() error {
	return e.Err
}
