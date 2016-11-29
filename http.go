package main

import (
	"net/http"
	"strconv"
)

func setProxyHeaders(req *http.Request) {
	req.Header.Set("Forwarded", forwarded(req))
}

func removeHopByHopHeaders(req *http.Request) {
	// Remove headers listed in the Connection header.
	for _, c := range req.Header["Connection"] {
		delete(req.Header, c)
	}

	// Remove hop-by-hop headers.
	for _, h := range [...]string{
		"Connection",
		"TE",
		"Transfer-Encoding",
		"Keep-Alive",
		"Proxy-Authorization",
		"Proxy-Authentication",
		"Upgrade",
	} {
		delete(req.Header, h)
	}

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
