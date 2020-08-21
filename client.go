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
)

// Response represents the response from a Gemini server.
type Response struct {
	Status int
	Meta   string
	Body   io.ReadCloser
	// Cert is the client or server cert received in the connection.
	// If you are the client, then it is the server cert, and vice versa.
	Cert *x509.Certificate
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
	// AllowInvalidStatuses means the client won't raise an error if a status
	// that is out of spec is returned.
	AllowInvalidStatuses bool
	// Insecure disables all TLS-based checks, use with caution.
	// It overrides all the variables above.
	Insecure bool
	// Timeout is equivalent to the Timeout field in net.Dialer.
	// It's the time it takes to form the initial connection.
	// The timeout of the DefaultClient is 15 seconds.
	Timeout time.Duration
}

var DefaultClient = &Client{Timeout: 15 * time.Second}

// Fetch a resource from a Gemini server with the given URL.
// It assumes port 1965 if no port is specified.
func (c *Client) Fetch(rawURL string) (*Response, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	host := parsedURL.Host
	if parsedURL.Port() == "" {
		host = net.JoinHostPort(parsedURL.Hostname(), "1965")
	}
	return c.FetchWithHost(host, rawURL)
}

// FetchWithHost fetches a resource from a Gemini server at the given host, with the given URL.
// This can be used for proxying, where the URL host and actual server don't match.
// It assumes the host is using port 1965 if no port number is provided.
func (c *Client) FetchWithHost(host, rawURL string) (*Response, error) {

	// URL checks
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	if len(rawURL) > 1024 {
		// Out of spec
		return nil, fmt.Errorf("url is too long")
	}

	// Add port to host if needed
	_, _, err = net.SplitHostPort(host)
	if err != nil {
		// Error likely means there's no port in the host
		host = net.JoinHostPort(host, "1965")
	}

	res := Response{}

	conn, err := c.connect(&res, host, parsedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the server: %v", err)
	}

	err = sendRequest(conn, parsedURL.String())
	if err != nil {
		conn.Close()
		return nil, err
	}

	err = getResponse(&res, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !c.AllowInvalidStatuses && !IsStatusValid(res.Status) {
		conn.Close()
		return nil, fmt.Errorf("invalid status code: %v", res.Status)
	}

	return &res, nil
}

// Fetch a resource from a Gemini server with the default client.
func Fetch(url string) (*Response, error) {
	return DefaultClient.Fetch(url)
}

// FetchWithHost fetches a resource from a Gemini server at the given host, with the default client.
// This can be used for proxying, where the URL host and actual server don't match.
func FetchWithHost(host, url string) (*Response, error) {
	return DefaultClient.FetchWithHost(host, url)
}

func (c *Client) connect(res *Response, host string, parsedURL *url.URL) (io.ReadWriteCloser, error) {
	conf := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // This must be set to allow self-signed certs
	}

	keylogfile := os.Getenv("SSLKEYLOGFILE")
	if keylogfile != "" {
		w, err := os.OpenFile(keylogfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err == nil {
			conf.KeyLogWriter = w
			defer w.Close()
		}
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: c.Timeout}, "tcp", host, conf)
	if err != nil {
		return conn, err
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
		if cert.Subject.CommonName != hostname && cert.VerifyHostname(hostname) != nil {
			return nil, fmt.Errorf("hostname does not verify")
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
	if len(meta) > 1024 {
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
