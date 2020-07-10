package gemini

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
)

// Request contains the data of the client request
type Request struct {
	URL *url.URL
}

// Handler is the interface a struct need to implement to be able to handle Gemini requests
type Handler interface {
	Handle(r Request) *Response
}

// ListenAndServe create a TCP server on the specified address and pass
// new connections to the given handler.
// Each request is handled in a separate goroutine.
func ListenAndServe(addr, certFile, keyFile string, handler Handler) error {
	if addr == "" {
		addr = "127.0.0.1:1965"
	}

	listener, err := listen(addr, certFile, keyFile)
	if err != nil {
		return err
	}

	err = serve(listener, handler)
	if err != nil {
		return err
	}

	err = listener.Close()
	if err != nil {
		return fmt.Errorf("failed to close the listener: %v", err)
	}

	return nil
}

func listen(addr, certFile, keyFile string) (net.Listener, error) {
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %v", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cer}}

	keylogfile := os.Getenv("SSLKEYLOGFILE")
	if keylogfile != "" {
		w, err := os.OpenFile(keylogfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err == nil {
			config.KeyLogWriter = w
			defer w.Close()
		}
	}

	ln, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	return ln, nil
}

func serve(listener net.Listener, handler Handler) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handleConnection(conn, handler)
	}
}

func handleConnection(conn io.ReadWriteCloser, handler Handler) {
	defer conn.Close()

	requestURL, err := getRequestURL(conn)
	if err != nil {
		// Return BadRequest (59) if there was URL parsing error
		writeResponse(conn, &Response{Status: StatusBadRequest, Meta: "Bad URL: " + err.Error()})
		return
	}

	request := Request{requestURL}
	response := handler.Handle(request)

	if response.Body != nil {
		defer response.Body.Close()
	}

	err = writeResponse(conn, response)
	if err != nil {
		return
	}
}

func getRequestURL(conn io.Reader) (*url.URL, error) {
	scanner := bufio.NewScanner(conn)
	if ok := scanner.Scan(); !ok {
		return nil, scanner.Err()
	}

	rawURL := strings.TrimSuffix(scanner.Text(), "\r\n")
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse request URL")
	}
	if parsedURL.User != nil {
		return nil, fmt.Errorf("userinfo not allowed in request URL")
	}
	if parsedURL.Scheme == "" {
		// Default scheme is gemini
		parsedURL.Scheme = "gemini"
	}

	return parsedURL, nil
}

func writeResponse(conn io.Writer, response *Response) error {
	_, err := fmt.Fprintf(conn, "%d %s\r\n", response.Status, response.Meta)
	if err != nil {
		return fmt.Errorf("failed to write header line to the client: %v", err)
	}

	if response.Body == nil {
		return nil
	}

	_, err = io.Copy(conn, response.Body)
	if err != nil {
		return fmt.Errorf("failed to write the response body to the client: %v", err)
	}

	return nil
}

// ErrorResponse create a response from the given error with the error string as the Meta field.
// If the error is of type gemini.Error, the status will be taken from the status field,
// otherwise it will default to StatusTemporaryFailure.
// If the error is nil, the function will panic.
func ErrorResponse(err error) *Response {
	if err == nil {
		panic("nil error is not a valid parameter")
	}

	if ge, ok := err.(Error); ok {
		return &Response{Status: ge.Status, Meta: ge.Error(), Body: nil}
	}

	return &Response{Status: StatusTemporaryFailure, Meta: err.Error(), Body: nil}
}
