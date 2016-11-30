package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/apex/log"
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
		return
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(url + ": " + res.Status)
		return
	}

	var list []struct {
		Address     string   `json:"Address"`
		ServicePort int      `json:"ServicePort"`
		ServiceTags []string `json:"ServiceTags"`
	}

	if err = json.NewDecoder(res.Body).Decode(&list); err != nil {
		return
	}

	srv = make([]service, 0, len(list))

	for _, s := range list {
		srv = append(srv, service{
			host: s.Address,
			port: s.ServicePort,
			tags: s.ServiceTags,
		})
	}

	log.WithFields(log.Fields{
		"name":     name,
		"url":      url,
		"status":   res.StatusCode,
		"services": len(srv),
	}).Info("consul service discovery")
	return
}
