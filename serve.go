package main

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func serveHTTP(w http.ResponseWriter, req *http.Request, rslv resolver, domain string) {
	connect := req.Header.Get("Connection")
	upgrade := req.Header.Get("Upgrade")

	host, port, _ := net.SplitHostPort(req.Host)

	if len(host) == 0 {
		host = req.Host
	}

	if !strings.HasSuffix(host, domain) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Resolve the hostname to a list of potential services.
	srv, err := rslv.resolve(host[:len(host)-len(domain)])

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(srv) == 0 {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	host = srv[0].host
	port = strconv.Itoa(srv[0].port)

	// Prepare the request to be forwarded to the service.
	setProxyHeaders(req)
	removeHopByHopHeaders(req)
	req.URL.Host = net.JoinHostPort(host, port)

	// If this is a request for a protocol upgrade we open a new tcp connection
	// to the service.
	if strings.EqualFold(connect, "Upgrade") && len(upgrade) != 0 {
		// TODO: support protocol upgrades
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	// Forward the request to the resolved hostname.
	res, err := http.DefaultTransport.RoundTrip(req)

	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// Configure the response header.
	h := w.Header()
	for k, v := range res.Header {
		h[k] = v
	}

	// Send the response.
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)

	// Done.
	res.Body.Close()
}
