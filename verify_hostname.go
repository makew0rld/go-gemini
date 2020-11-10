// Source: https://git.sr.ht/~adnano/go-gemini/tree/f6b0443a6262d17f90b4e75cf5ae37577db7f897/vendor.go
// No code changes were made.

// Hostname verification code from the crypto/x509 package.
// Modified to allow Common Names in the short term, until new certificates
// can be issued with SANs.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-GO file.

package gemini

import (
	"crypto/x509"
	"net"
	"strings"
	"unicode/utf8"
)

var oidExtensionSubjectAltName = []int{2, 5, 29, 17}

func hasSANExtension(c *x509.Certificate) bool {
	for _, e := range c.Extensions {
		if e.Id.Equal(oidExtensionSubjectAltName) {
			return true
		}
	}
	return false
}

func validHostnamePattern(host string) bool { return validHostname(host, true) }
func validHostnameInput(host string) bool   { return validHostname(host, false) }

// validHostname reports whether host is a valid hostname that can be matched or
// matched against according to RFC 6125 2.2, with some leniency to accommodate
// legacy values.
func validHostname(host string, isPattern bool) bool {
	if !isPattern {
		host = strings.TrimSuffix(host, ".")
	}
	if len(host) == 0 {
		return false
	}

	for i, part := range strings.Split(host, ".") {
		if part == "" {
			// Empty label.
			return false
		}
		if isPattern && i == 0 && part == "*" {
			// Only allow full left-most wildcards, as those are the only ones
			// we match, and matching literal '*' characters is probably never
			// the expected behavior.
			continue
		}
		for j, c := range part {
			if 'a' <= c && c <= 'z' {
				continue
			}
			if '0' <= c && c <= '9' {
				continue
			}
			if 'A' <= c && c <= 'Z' {
				continue
			}
			if c == '-' && j != 0 {
				continue
			}
			if c == '_' {
				// Not a valid character in hostnames, but commonly
				// found in deployments outside the WebPKI.
				continue
			}
			return false
		}
	}

	return true
}

// commonNameAsHostname reports whether the Common Name field should be
// considered the hostname that the certificate is valid for. This is a legacy
// behavior, disabled by default or if the Subject Alt Name extension is present.
//
// It applies the strict validHostname check to the Common Name field, so that
// certificates without SANs can still be validated against CAs with name
// constraints if there is no risk the CN would be matched as a hostname.
// See NameConstraintsWithoutSANs and issue 24151.
func commonNameAsHostname(c *x509.Certificate) bool {
	return !hasSANExtension(c) && validHostnamePattern(c.Subject.CommonName)
}

func matchExactly(hostA, hostB string) bool {
	if hostA == "" || hostA == "." || hostB == "" || hostB == "." {
		return false
	}
	return toLowerCaseASCII(hostA) == toLowerCaseASCII(hostB)
}

func matchHostnames(pattern, host string) bool {
	pattern = toLowerCaseASCII(pattern)
	host = toLowerCaseASCII(strings.TrimSuffix(host, "."))

	if len(pattern) == 0 || len(host) == 0 {
		return false
	}

	patternParts := strings.Split(pattern, ".")
	hostParts := strings.Split(host, ".")

	if len(patternParts) != len(hostParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if i == 0 && patternPart == "*" {
			continue
		}
		if patternPart != hostParts[i] {
			return false
		}
	}

	return true
}

// toLowerCaseASCII returns a lower-case version of in. See RFC 6125 6.4.1. We use
// an explicitly ASCII function to avoid any sharp corners resulting from
// performing Unicode operations on DNS labels.
func toLowerCaseASCII(in string) string {
	// If the string is already lower-case then there's nothing to do.
	isAlreadyLowerCase := true
	for _, c := range in {
		if c == utf8.RuneError {
			// If we get a UTF-8 error then there might be
			// upper-case ASCII bytes in the invalid sequence.
			isAlreadyLowerCase = false
			break
		}
		if 'A' <= c && c <= 'Z' {
			isAlreadyLowerCase = false
			break
		}
	}

	if isAlreadyLowerCase {
		return in
	}

	out := []byte(in)
	for i, c := range out {
		if 'A' <= c && c <= 'Z' {
			out[i] += 'a' - 'A'
		}
	}
	return string(out)
}

// verifyHostname returns nil if c is a valid certificate for the named host.
// Otherwise it returns an error describing the mismatch.
//
// IP addresses can be optionally enclosed in square brackets and are checked
// against the IPAddresses field. Other names are checked case insensitively
// against the DNSNames field. If the names are valid hostnames, the certificate
// fields can have a wildcard as the left-most label.
//
// The legacy Common Name field is ignored unless it's a valid hostname, the
// certificate doesn't have any Subject Alternative Names, and the GODEBUG
// environment variable is set to "x509ignoreCN=0". Support for Common Name is
// deprecated will be entirely removed in the future.
func verifyHostname(c *x509.Certificate, h string) error {
	// IP addresses may be written in [ ].
	candidateIP := h
	if len(h) >= 3 && h[0] == '[' && h[len(h)-1] == ']' {
		candidateIP = h[1 : len(h)-1]
	}
	if ip := net.ParseIP(candidateIP); ip != nil {
		// We only match IP addresses against IP SANs.
		// See RFC 6125, Appendix B.2.
		for _, candidate := range c.IPAddresses {
			if ip.Equal(candidate) {
				return nil
			}
		}
		return x509.HostnameError{c, candidateIP}
	}

	names := c.DNSNames
	if commonNameAsHostname(c) {
		names = []string{c.Subject.CommonName}
	}

	candidateName := toLowerCaseASCII(h) // Save allocations inside the loop.
	validCandidateName := validHostnameInput(candidateName)

	for _, match := range names {
		// Ideally, we'd only match valid hostnames according to RFC 6125 like
		// browsers (more or less) do, but in practice Go is used in a wider
		// array of contexts and can't even assume DNS resolution. Instead,
		// always allow perfect matches, and only apply wildcard and trailing
		// dot processing to valid hostnames.
		if validCandidateName && validHostnamePattern(match) {
			if matchHostnames(match, candidateName) {
				return nil
			}
		} else {
			if matchExactly(match, candidateName) {
				return nil
			}
		}
	}

	return x509.HostnameError{c, h}
}
