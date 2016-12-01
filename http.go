package main

import (
	"io"
	"net/http"
	"strconv"
)

type bodyReader struct {
	io.Reader
	n int
}

func (r *bodyReader) Read(b []byte) (n int, err error) {
	if n, err = r.Reader.Read(b); n > 0 {
		r.n += n
	}
	return
}

func (r *bodyReader) Close() error {
	return nil // don't close request bodies so we can do retries
}

func idempotent(method string) bool {
	switch method {
	case "GET", "HEAD", "PUT", "DELETE", "OPTIONS":
		return true
	}
	return false
}

func copyHeader(to http.Header, from http.Header) {
	for field, value := range from {
		to[field] = value
	}
}

func clearConnectionFields(hdr http.Header) {
	for _, field := range hdr["Connection"] {
		hdr.Del(field)
	}
}

func clearHopByHopFields(hdr http.Header) {
	for _, field := range [...]string{
		"Connection",
		"TE",
		"Transfer-Encoding",
		"Keep-Alive",
		"Proxy-Authorization",
		"Proxy-Authentication",
		"Upgrade",
	} {
		hdr.Del(field)
	}
}

func clearRequestMetadata(req *http.Request) {
	// These fields are populated by the standard http server implementation but
	// don't make sense or are invalid to set on client requests.
	req.TransferEncoding = nil
	req.Close = false
	req.RequestURI = ""
}

func forwarded(req *http.Request) string {
	// TODO: combine with previous Forwarded or X-Forwarded-For header.
	return "for=" + quote(req.RemoteAddr) + ";host=" + quote(req.Host) + ";proto=http"
}

func quote(s string) string {
	// TODO: https://tools.ietf.org/html/rfc7230#section-3.2.6
	return strconv.QuoteToASCII(s)
}
