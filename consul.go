package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// The consulResolver is a resolver implementation that uses a consul agent to
// lookup registered services.
type consulResolver struct {
	address string
}

func (r consulResolver) resolve(name string) (srv []service, err error) {
	var res *http.Response
	var url = r.address + "/v1/catalog/service/" + name

	switch {
	case strings.HasPrefix(url, "http://"):
	case strings.HasPrefix(url, "https://"):
	default:
		url = "http://" + url
	}

	if res, err = http.Get(url); err != nil {
		err = errors.Wrap(err, url)
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(url + ": " + res.Status)
		return
	}

	list := []struct {
		Address     string
		ServicePort int
		ServiceTags []string
	}{}

	if err = json.NewDecoder(res.Body).Decode(&list); err != nil {
		return
	}

	srv = make([]service, len(list))

	for i, s := range list {
		srv[i] = service{
			host: s.Address,
			port: s.ServicePort,
			tags: s.ServiceTags,
		}
	}

	return
}
