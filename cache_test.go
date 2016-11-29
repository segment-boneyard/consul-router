package main

import (
	"reflect"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	services := serviceMap{
		"host-1": []service{{"host-1", 1000, nil}},
		"host-2": []service{{"host-2", 2000, nil}},
		"host-3": []service{{"host-3", 3000, []string{"A", "B", "C"}}},
	}

	cache := newCache(cacheConfig{
		timeout: time.Second,
		size:    2,
		rslv:    services,
	})

	for _, name := range []string{"host-1", "host-2", "host-3"} {
		t.Run(name, func(t *testing.T) {
			srv, err := cache.resolve(name)
			if err != nil {
				t.Error(err)
			} else if !reflect.DeepEqual(srv, services[name]) {
				t.Errorf("%#v != %#v", srv, services[name])
			}
		})
	}

	t.Run("missing", func(t *testing.T) {
		srv, err := cache.resolve("")
		if err != nil {
			t.Error(err)
		} else if len(srv) != 0 {
			t.Errorf("%#v", srv)
		}
	})
}
