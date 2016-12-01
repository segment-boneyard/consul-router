package main

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apex/log"
)

// The server type is a http handler that proxies requests and uses a resolver
// to lookup the address to which it should send the requests.
type server struct {
	domain string
	rslv   resolver
}

func newServer(domain string, rslv resolver) *server {
	return &server{
		domain: domain,
		rslv:   rslv,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// If this is a request for a protocol upgrade we open a new tcp connection
	// to the service.
	if len(req.Header.Get("Upgrade")) != 0 {
		// TODO: support protocol upgrades
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	if !strings.HasSuffix(req.Host, s.domain) {
		w.WriteHeader(http.StatusServiceUnavailable)
		log.WithFields(log.Fields{
			"status": http.StatusServiceUnavailable,
			"host":   req.Host,
			"domain": s.domain,
			"reason": "the requested host doesn't belong to the domain served by the router",
		}).Error(http.StatusText(http.StatusServiceUnavailable))
		return
	}

	host := req.Host
	name := host[:len(host)-len(s.domain)]
	clearConnectionFields(req.Header)
	clearHopByHopFields(req.Header)
	clearRequestMetadata(req)

	// Forward the request to the resolved hostname, connection errors are
	// retried on idempotent methods, only if no bytes of the body have been
	// transfered yet.
	const maxAttempts = 10
	var res *http.Response

	body := &bodyReader{Reader: req.Body}
	req.Body = body

	for attempt := 0; true; attempt++ {
		srv, err := s.rslv.resolve(name)

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

		// Prepare the request to be forwarded to the service.
		req.Host = srv[0].host
		req.URL.Scheme = "http"
		req.URL.Host = net.JoinHostPort(srv[0].host, strconv.Itoa(srv[0].port))
		req.Header.Set("Forwarded", forwarded(req))
		req.Header.Set("Host", req.Host)

		if res, err = http.DefaultTransport.RoundTrip(req); err == nil {
			break // success
		}

		if attempt < maxAttempts && body.n == 0 && idempotent(req.Method) {
			// Backoff: 0ms, 10ms, 40ms, 90ms ... 1000ms
			time.Sleep(time.Duration(attempt*attempt) * 10 * time.Millisecond)
			continue
		}

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
	copyHeader(h, res.Header)
	clearConnectionFields(h)
	clearHopByHopFields(h)

	// Send the response.
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	res.Body.Close()
}
