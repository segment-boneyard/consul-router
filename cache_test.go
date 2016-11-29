package main

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	services := serviceMap{
		"host-1": []service{{"host-1", 1000, nil}},
		"host-2": []service{{"host-2", 2000, nil}},
		"host-3": []service{{"host-3", 3000, []string{"A", "B", "C"}}},
	}

	cache := cached(time.Second, services)

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

func BenchmarkCache(b *testing.B) {
	for _, size := range [...]int{1, 10, 100, 1000} {
		name, services := make([]string, size), make(serviceMap, size)

		for i := 0; i != size; i++ {
			name := fmt.Sprintf("host-%d", i+1)
			names[i] = name
			services[name] = []service{{name, 4242, nil}}
		}

		cache := cached(1*time.Minute, services)

		b.Run(strconv.Itoa(size), func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				cache.resolve(names[i%len(names)])
			}
		})
	}
}
