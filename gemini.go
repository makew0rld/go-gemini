package gemini

import "fmt"

// Gemini status codes as defined in the Gemini spec Appendix 1.
const (
	StatusInput = 10

	StatusSuccess                              = 20
	StatusSuccessEndOfClientCertificateSession = 21

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

	StatusClientCertificateRequired     = 60
	StatusTransientCertificateRequested = 61
	StatusAuthorisedCertificateRequired = 62
	StatusCertificateNotAccepted        = 63
	StatusFutureCertificateRejected     = 64
	StatusExpiredCertificateRejected    = 65
)

// SimplifyStatus simplify the response status by omiting the detailed second digit of the status code.
func SimplifyStatus(status int) int {
	return (status / 10) * 10
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
