package gemini

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// Response represent the response from a Gemini server.
type Response struct {
	Status int
	Meta   string
	Body   io.ReadCloser
}

type header struct {
	status int
	meta   string
}

// Fetch a resource from a Gemini server with the given URL
func Fetch(url string) (res Response, err error) {
	conn, err := connectByURL(url)
	if err != nil {
		return Response{}, fmt.Errorf("failed to connect to the server: %v", err)
	}

	err = sendRequest(conn, url)
	if err != nil {
		conn.Close()
		return Response{}, err
	}

	return getResponse(conn)
}

func connectByURL(rawURL string) (io.ReadWriteCloser, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse given URL: %v", err)
	}

	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	return tls.Dial("tcp", parsedURL.Host, conf)
}

func sendRequest(conn io.Writer, requestURL string) error {
	_, err := fmt.Fprintf(conn, "%s\r\n", requestURL)
	if err != nil {
		return fmt.Errorf("could not send request to the server: %v", err)
	}

	return nil
}

func getResponse(conn io.ReadCloser) (Response, error) {
	header, err := getHeader(conn)
	if err != nil {
		conn.Close()
		return Response{}, fmt.Errorf("failed to get header: %v", err)
	}

	return Response{header.status, header.meta, conn}, nil
}

func getHeader(conn io.Reader) (header, error) {
	line, err := readHeader(conn)
	if err != nil {
		return header{}, fmt.Errorf("failed to read header: %v", err)
	}

	fields := strings.Fields(string(line))
	status, err := strconv.Atoi(fields[0])
	if err != nil {
		return header{}, fmt.Errorf("unexpected status value %v: %v", fields[0], err)
	}

	meta := strings.Join(fields[1:], " ")

	return header{status, meta}, nil
}

func readHeader(conn io.Reader) ([]byte, error) {
	var line []byte
	delim := []byte("\r\n")
	// A small buffer is inefficient but the maximum length of the header is small so it's okay
	buf := make([]byte, 1)

	for {
		_, err := conn.Read(buf)
		if err != nil {
			return []byte{}, err
		}

		line = append(line, buf...)
		if bytes.HasSuffix(line, delim) {
			return line[:len(line)-len(delim)], nil
		}
	}
}
