package gemini

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

func punycodeHost(host string) (string, error) {
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		// Likely means no port
		hostname = host
		port = ""
	}

	if net.ParseIP(hostname) != nil {
		// Hostname is IP address, not domain
		return host, nil
	}
	pc, err := idna.ToASCII(hostname)
	if err != nil {
		return host, err
	}
	if port == "" {
		return pc, nil
	}
	return net.JoinHostPort(pc, port), nil
}

func punycodeHostFromURL(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return punycodeHost(parsed.Host)
}

// GetPunycodeURL takes a full URL that potentially has Unicode in the
// domain name, and returns a URL with the domain punycoded.
func GetPunycodeURL(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", nil
	}
	host, err := punycodeHostFromURL(u)
	if err != nil {
		return "", err
	}
	parsed.Host = host
	return parsed.String(), nil
}

// Response represents the response from a Gemini server.
type Response struct {
	Status int
	Meta   string
	Body   io.ReadCloser
	Cert   *x509.Certificate
	conn   net.Conn
}

type header struct {
	status int
	meta   string
}

type Client struct {
	// NoTimeCheck allows connections with expired or future certs if set to true.
	NoTimeCheck bool

	// NoHostnameCheck allows connections when the cert doesn't match the
	// requested hostname or IP.
	NoHostnameCheck bool

	// Insecure disables all TLS-based checks, use with caution.
	// It overrides all the variables above.
	Insecure bool

	// AllowInvalidStatuses means the client won't raise an error if a status
	// that is out of spec is returned.
	AllowInvalidStatuses bool

	// ConnectTimeout is equivalent to the Timeout field in net.Dialer.
	// It's the max amount of time allowed for the initial connection/handshake.
	// The timeout of the DefaultClient is 15 seconds.
	//
	// If ReadTimeout is not set, then this value is also used to time out on getting
	// the header after the connection is made.
	ConnectTimeout time.Duration

	// ReadTimeout is the max amount of time reading to a server can take.
	// This should not be set if you want to support streams.
	// It is equivalent to net.Conn.SetDeadline, see that func for more documentation.
	//
	// For example, if this is set to 30 seconds, then no more reading from the connection
	// can happen 30 seconds after the initial handshake.
	ReadTimeout time.Duration
}

var DefaultClient = &Client{ConnectTimeout: 15 * time.Second}

// getHost returns a full host for the given URL, always including a port.
// It also punycodes the host, in case it contains Unicode.
func getHost(parsedURL *url.URL) string {
	host, _ := punycodeHostFromURL(parsedURL.String())
	if parsedURL.Port() == "" {
		host = net.JoinHostPort(parsedURL.Hostname(), "1965")
	}
	return host
}

// SetReadTimeout changes the read timeout after the connection has been made.
// You can set it to 0 or less to disable the timeout. Otherwise, the duration
// is relative to the time the function was called.
func (r *Response) SetReadTimeout(d time.Duration) error {
	if d <= 0 {
		return r.conn.SetDeadline(time.Time{})
	}
	return r.conn.SetDeadline(time.Now().Add(d))
}

// TODO: apply punycoding to hosts

// Fetch a resource from a Gemini server with the given URL.
// It assumes port 1965 if no port is specified.
func (c *Client) Fetch(rawURL string) (*Response, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	return c.FetchWithHost(getHost(parsedURL), rawURL)
}

// FetchWithHost fetches a resource from a Gemini server at the given host, with the given URL.
// This can be used for proxying, where the URL host and actual server don't match.
// It assumes the host is using port 1965 if no port number is provided.
func (c *Client) FetchWithHost(host, rawURL string) (*Response, error) {
	// Call with empty PEM bytes to skip using a cert
	return c.FetchWithHostAndCert(host, rawURL, []byte{}, []byte{})
}

// FetchWithCert fetches a resource from a Gemini server with the given URL.
// It allows you to provide the bytes of a PEM encoded block for a client
// certificate and its key. This allows you to make requests using client
// certs.
//
// It assumes port 1965 if no port is specified.
func (c *Client) FetchWithCert(rawURL string, certPEM, keyPEM []byte) (*Response, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	// Call with empty PEM bytes to skip using a cert
	return c.FetchWithHostAndCert(getHost(parsedURL), rawURL, certPEM, keyPEM)
}

// FetchWithHostAndCert combines FetchWithHost and FetchWithCert.
func (c *Client) FetchWithHostAndCert(host, rawURL string, certPEM, keyPEM []byte) (*Response, error) {
	u, err := GetPunycodeURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("error when punycoding URL: %v", err)
	}
	parsedURL, _ := url.Parse(u)

	if len(u) > URLMaxLength {
		// Out of spec
		return nil, fmt.Errorf("url is too long")
	}

	// Add port to host if needed
	_, _, err = net.SplitHostPort(host)
	if err != nil {
		// Error likely means there's no port in the host
		host = net.JoinHostPort(host, "1965")
	}
	ogHost := host
	host, err = punycodeHost(host)
	if err != nil {
		return nil, fmt.Errorf("failed to punycode host %s: %v", ogHost, err)
	}

	// Build tls.Certificate
	var cert tls.Certificate
	if len(certPEM) == 0 && len(keyPEM) == 0 {
		// Cert bytes were intentionally left empty
		cert = tls.Certificate{}
	} else {
		cert, err = tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cert/key PEM: %v", err)
		}
	}

	res := Response{}

	// Connect

	start := time.Now()
	conn, err := c.connect(&res, host, parsedURL, cert)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the server: %v", err)
	}

	// Send request

	if c.ReadTimeout == 0 && c.ConnectTimeout != 0 {
		// No r/w timeout, so a timeout for sending the request must be set
		conn.SetDeadline(start.Add(c.ConnectTimeout))
	}
	err = sendRequest(conn, u)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if c.ReadTimeout == 0 && c.ConnectTimeout != 0 {
		// Undo deadline
		conn.SetDeadline(time.Time{})
	}

	// Get header

	if c.ReadTimeout == 0 && c.ConnectTimeout != 0 {
		// No r/w timeout, so a timeout for getting the header
		conn.SetDeadline(start.Add(c.ConnectTimeout))
	}
	err = getResponse(&res, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if c.ReadTimeout == 0 && c.ConnectTimeout != 0 {
		// Undo deadline
		conn.SetDeadline(time.Time{})
	}

	// Check status code
	if !c.AllowInvalidStatuses && !IsStatusValid(res.Status) {
		conn.Close()
		return nil, fmt.Errorf("invalid status code: %v", res.Status)
	}

	return &res, nil
}

// Fetch a resource from a Gemini server with the given URL.
// It assumes port 1965 if no port is specified.
func Fetch(url string) (*Response, error) {
	return DefaultClient.Fetch(url)
}

// FetchWithCert fetches a resource from a Gemini server with the given URL.
// It allows you to provide the bytes of a PEM encoded block for a client
// certificate and its key. This allows you to make requests using client
// certs.
//
// It assumes port 1965 if no port is specified.
func FetchWithCert(url string, certPEM, keyPEM []byte) (*Response, error) {
	return DefaultClient.FetchWithCert(url, certPEM, keyPEM)
}

// FetchWithHost fetches a resource from a Gemini server at the given host, with the given URL.
// This can be used for proxying, where the URL host and actual server don't match.
// It assumes the host is using port 1965 if no port number is provided.
func FetchWithHost(host, url string) (*Response, error) {
	return DefaultClient.FetchWithHost(host, url)
}

// FetchWithHostAndCert combines FetchWithHost and FetchWithCert.
func FetchWithHostAndCert(host, url string, certPEM, keyPEM []byte) (*Response, error) {
	return DefaultClient.FetchWithHostAndCert(host, url, certPEM, keyPEM)
}

func (c *Client) connect(res *Response, host string, parsedURL *url.URL, clientCert tls.Certificate) (net.Conn, error) {
	conf := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // This must be set to allow self-signed certs
	}
	if clientCert.Certificate != nil {
		// There is data, not an empty struct
		conf.Certificates = []tls.Certificate{clientCert}
	}

	// Support logging TLS keys for debugging - See PR #5
	keylogfile := os.Getenv("SSLKEYLOGFILE")
	if keylogfile != "" {
		w, err := os.OpenFile(keylogfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err == nil {
			conf.KeyLogWriter = w
			defer w.Close()
		}
	}

	// Dialer timeout for handshake
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: c.ConnectTimeout}, "tcp", host, conf)
	res.conn = conn
	if err != nil {
		return conn, err
	}

	if c.ReadTimeout != 0 {
		conn.SetDeadline(time.Now().Add(c.ReadTimeout))
	}

	cert := conn.ConnectionState().PeerCertificates[0]
	res.Cert = cert

	if c.Insecure {
		return conn, nil
	}

	// Verify hostname
	if !c.NoHostnameCheck {
		// Cert hostname has to match connection host, not request host
		hostname, _, _ := net.SplitHostPort(host)

		if err := verifyHostname(cert, hostname); err != nil {
			// Try with Unicode version
			uniHost, uniErr := idna.ToUnicode(hostname)
			err2 := verifyHostname(cert, uniHost)
			if uniErr != nil {
				return nil, fmt.Errorf("punycoded hostname does not verify and could not be converted to Unicode: %v", err)
			}
			if err2 != nil {
				return nil, fmt.Errorf("hostname does not verify: %v", err2)
			}
			return nil, fmt.Errorf("hostname does not verify: %v", err)
		}
	}
	// Verify expiry
	if !c.NoTimeCheck {
		if cert.NotBefore.After(time.Now()) {
			return nil, fmt.Errorf("server cert is for the future")
		} else if cert.NotAfter.Before(time.Now()) {
			return nil, fmt.Errorf("server cert is expired")
		}
	}

	return conn, nil
}

func sendRequest(conn io.Writer, requestURL string) error {
	_, err := fmt.Fprintf(conn, "%s\r\n", requestURL)
	if err != nil {
		return fmt.Errorf("could not send request to the server: %v", err)
	}
	return nil
}

func getResponse(res *Response, conn io.ReadCloser) error {
	header, err := getHeader(conn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to get header: %v", err)
	}

	res.Status = header.status
	res.Meta = header.meta
	res.Body = conn
	return nil
}

func getHeader(conn io.Reader) (header, error) {
	line, err := readHeader(conn)
	if err != nil {
		return header{}, fmt.Errorf("failed to read header: %v", err)
	}

	fields := strings.Fields(string(line))
	if len(fields) < 2 && line[len(line)-1] != ' ' {
		return header{}, fmt.Errorf("header not formatted correctly")
	}

	status, err := strconv.Atoi(fields[0])
	if err != nil {
		return header{}, fmt.Errorf("unexpected status value %v: %v", fields[0], err)
	}

	var meta string
	if len(line) <= 3 {
		meta = ""
	} else {
		meta = string(line)[len(fields[0])+1:]
	}
	if len(meta) > MetaMaxLength {
		return header{}, fmt.Errorf("meta string is too long")
	}

	return header{status, meta}, nil
}

func readHeader(conn io.Reader) ([]byte, error) {
	var line []byte
	delim := []byte("\r\n")
	// A small buffer is inefficient but the maximum length of the header is small so it's okay
	buf := make([]byte, 1)

	for {
		n, err := conn.Read(buf)
		if err == io.EOF && n <= 0 {
			return []byte{}, err
		} else if err != nil && err != io.EOF {
			return []byte{}, err
		}

		line = append(line, buf...)
		if bytes.HasSuffix(line, delim) {
			return line[:len(line)-len(delim)], nil
		}
	}
}
