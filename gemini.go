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

var statusText = map[int]string{
	StatusInput:          "Input",
	StatusSensitiveInput: "Sensitive Input",

	StatusSuccess: "Success",

	// StatusRedirect:       "Redirect - Temporary"
	StatusRedirectTemporary: "Redirect - Temporary",
	StatusRedirectPermanent: "Redirect - Permanent",

	StatusTemporaryFailure: "Temporary Failure",
	StatusUnavailable:      "Server Unavailable",
	StatusCGIError:         "CGI Error",
	StatusProxyError:       "Proxy Error",
	StatusSlowDown:         "Slow Down",

	StatusPermanentFailure:    "Permanent Failure",
	StatusNotFound:            "Not Found",
	StatusGone:                "Gone",
	StatusProxyRequestRefused: "Proxy Request Refused",
	StatusBadRequest:          "Bad Request",

	StatusClientCertificateRequired: "Client Certificate Required",
	StatusCertificateNotAuthorised:  "Certificate Not Authorised",
	StatusCertificateNotValid:       "Certificate Not Valid",
}

// StatusText returns a text for the Gemini status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	return statusText[code]
}

// SimplifyStatus simplify the response status by ommiting the detailed second digit of the status code.
func SimplifyStatus(status int) int {
	return (status / 10) * 10
}

// IsStatusValid checks whether an int status is covered by the spec.
// Note that:
//     A client SHOULD deal with undefined status codes
//     between '10' and '69' per the default action of the initial digit.
func IsStatusValid(status int) bool {
	_, found := statusText[status]
	return found
}

// StatusInRange returns true if the status has a valid first digit.
// This means it can be handled even if it's not defined by the spec,
// because it has a known category
func StatusInRange(status int) bool {
	if status < 10 && status > 69 {
		return false
	}
	return true
}

// CleanStatus returns the status code as is, unless it's invalid but still in range
// Then it returns the status code with the second digit zeroed. So 51 returns 51,
// but 22 returns 20.
//
// This corresponds with the spec:
//     A client SHOULD deal with undefined status codes
//     between '10' and '69' per the default action of the initial digit.
func CleanStatus(status int) int {
	// All the functions come together!
	if !IsStatusValid(status) && StatusInRange(status) {
		return SimplifyStatus(status)
	}
	return status
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
