package main

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/apex/log"
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
		log.WithFields(log.Fields{
			"status": http.StatusServiceUnavailable,
			"host":   host,
			"domain": domain,
			"reason": "the requested host doesn't belong to the domain served by the router",
		}).Error(http.StatusText(http.StatusServiceUnavailable))
		return
	}

	// Resolve the hostname to a list of potential services.
	srv, err := rslv.resolve(host[:len(host)-len(domain)])

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.WithFields(log.Fields{
			"status": http.StatusInternalServerError,
			"host":   host,
			"error":  err,
			"reason": "an error was returned by the resolver",
		}).Error(http.StatusText(http.StatusInternalServerError))
		return
	}

	if len(srv) == 0 {
		w.WriteHeader(http.StatusBadGateway)
		log.WithFields(log.Fields{
			"status": http.StatusBadGateway,
			"host":   host,
			"reason": "no service returned by the resolver",
		}).Error(http.StatusText(http.StatusBadGateway))
		return
	}

	host = srv[0].host
	port = strconv.Itoa(srv[0].port)

	// Prepare the request to be forwarded to the service.
	req.Host = net.JoinHostPort(host, port)
	req.URL.Scheme = "http"
	req.URL.Host = req.Host
	req.Header.Set("Forwarded", forwarded(req))
	req.Header.Set("Host", req.Host)
	removeHopByHopHeaders(req)

	if len(req.URL.Scheme) == 0 {
		req.URL.Scheme = "http"
	}

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
		log.WithFields(log.Fields{
			"status": http.StatusBadGateway,
			"host":   host,
			"error":  err,
			"reason": "forwarding the request to the service returned an error",
		}).Error(http.StatusText(http.StatusBadGateway))
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
	res.Body.Close()
}
