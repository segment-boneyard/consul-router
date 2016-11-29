package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestConsul(t *testing.T) {
	services := serviceMap{
		"host-1": []service{{"host-1", 1000, nil}},
		"host-2": []service{{"host-2", 2000, nil}},
		"host-3": []service{{"host-3", 3000, []string{"A", "B", "C"}}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, "/v1/catalog/service/") {
			t.Error("invalid path:", req.URL.Path)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		name := path.Base(req.URL.Path)
		srv, _ := services[name]
		ret := []map[string]interface{}{}

		for _, s := range srv {
			ret = append(ret, map[string]interface{}{
				"Address":     s.host,
				"ServicePort": s.port,
				"ServiceTags": s.tags,
			})
		}

		json.NewEncoder(res).Encode(ret)
	}))
	defer server.Close()

	rslv := consulResolver{
		address: server.URL,
	}

	for _, name := range []string{"host-1", "host-2", "host-3"} {
		t.Run(name, func(t *testing.T) {
			srv, err := rslv.resolve(name)
			if err != nil {
				t.Error(err)
			} else if !reflect.DeepEqual(srv, services[name]) {
				t.Errorf("%#v != %#v", srv, services[name])
			}
		})
	}

	t.Run("missing", func(t *testing.T) {
		srv, err := rslv.resolve("")
		if err != nil {
			t.Error(err)
		} else if len(srv) != 0 {
			t.Errorf("%#v", srv)
		}
	})

}
