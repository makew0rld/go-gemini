// Package gemini provides an easy interface to create client and servers that
// speak the Gemini protocol.
//
// At the moment, this library is client-side only, and support is not guaranteed.
// It is mostly a personal library.
//
// It will automatically handle URLs that have IDNs in them, ie domains with Unicode.
// It will convert to punycode for DNS and for sending to the server, but accept
// certs with either punycode or Unicode as the hostname.
//
// This also applies to hosts, for functions where a host can be passed specifically.
package gemini
