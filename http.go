package main

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apex/log"
)

// The httpServer type is a http handler that proxies requests and uses a
// resolver to lookup the address to which it should send the requests.
type httpServer struct {
	domain    string
	blacklist *blacklist
	cache     *cache
	rslv      resolver
	join      sync.WaitGroup
	stop      uint32 // atomic flag
}

type httpServerConfig struct {
	stop         <-chan struct{}
	done         chan<- struct{}
	rslv         resolver
	domain       string
	prefer       string
	cacheTimeout time.Duration
}

func newHttpServer(config httpServerConfig) *httpServer {
	c := cached(config.cacheTimeout, config.rslv)
	b := blacklisted(config.cacheTimeout, c)
	s := &httpServer{
		domain:    config.domain,
		blacklist: b,
		cache:     c,
		rslv:      preferred(config.prefer, b),
	}

	go func(s *httpServer, stop <-chan struct{}, done chan<- struct{}) {
		// Wait for a stop signal, when it arrives the server is marked for
		// graceful shutdown and waits for in-flight requests to complete.
		// Note that this is not a perfect graceful shutdown and there may still
		// be some race conditions where requests are dropped but it's the best
		// we can do considering the current net/http API.
		<-stop
		s.setStopped()
		s.join.Wait()
		close(done)
	}(s, config.stop, config.done)

	return s
}

func (s *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.join.Add(1)
	defer s.join.Done()

	// When the server is stopped we break here returning a 503.
	if s.stopped() {
		w.Header().Add("Connection", "close")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

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
			"reason": http.StatusText(http.StatusServiceUnavailable),
			"host":   req.Host,
			"domain": s.domain,
		}).Error("the requested host doesn't belong to the domain served by the router")
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

	body := &httpBodyReader{Reader: req.Body}
	req.Body = body

	for attempt := 0; true; attempt++ {
		srv, err := s.rslv.resolve(name)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.WithFields(log.Fields{
				"status": http.StatusInternalServerError,
				"reason": http.StatusText(http.StatusInternalServerError),
				"host":   host,
				"error":  err,
			}).Error("an error was returned by the resolver")
			return
		}

		if len(srv) == 0 {
			w.WriteHeader(http.StatusBadGateway)
			log.WithFields(log.Fields{
				"status": http.StatusBadGateway,
				"reason": http.StatusText(http.StatusBadGateway),
				"host":   host,
			}).Error("no service returned by the resolver")
			return
		}

		// Prepare the request to be forwarded to the service.
		address := net.JoinHostPort(srv[0].host, strconv.Itoa(srv[0].port))
		req.URL.Scheme = "http"
		req.URL.Host = address
		req.Header.Set("Forwarded", forwarded(req))

		if res, err = http.DefaultTransport.RoundTrip(req); err == nil {
			break // success
		}

		if attempt < maxAttempts && body.n == 0 && idempotent(req.Method) {
			// Adding the host to the list of black-listed addresses so it
			// doesn't get picked up again for the next retries.
			s.blacklist.add(address)
			log.WithFields(log.Fields{
				"host":    host,
				"address": address,
				"error":   err,
			}).Warn("black-listing failing service")

			// Backoff: 0ms, 10ms, 40ms, 90ms ... 1000ms
			time.Sleep(time.Duration(attempt*attempt) * 10 * time.Millisecond)
			continue
		}

		w.WriteHeader(http.StatusBadGateway)
		log.WithFields(log.Fields{
			"status": http.StatusBadGateway,
			"reason": http.StatusText(http.StatusBadGateway),
			"host":   host,
			"error":  err,
		}).Error("forwarding the request to the service returned an error")
		return
	}

	// Configure the response header, remove headers that were not directed at
	// the client, add 'Connection: close' if the server is terminating.
	hdr := w.Header()
	copyHeader(hdr, res.Header)
	clearConnectionFields(hdr)
	clearHopByHopFields(hdr)

	if s.stopped() {
		hdr.Add("Connection", "close")
	}

	// Send the response.
	w.WriteHeader(res.StatusCode)
	copyBytes(w, res.Body)
	res.Body.Close()
}

func (s *httpServer) setStopped() {
	atomic.StoreUint32(&s.stop, 1)
}

func (s *httpServer) stopped() bool {
	return atomic.LoadUint32(&s.stop) != 0
}

type httpBodyReader struct {
	io.Reader
	n int
}

func (r *httpBodyReader) Read(b []byte) (n int, err error) {
	if n, err = r.Reader.Read(b); n > 0 {
		r.n += n
	}
	return
}

func (r *httpBodyReader) Close() error {
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
