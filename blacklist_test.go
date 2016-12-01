package main

import (
	"reflect"
	"testing"
	"time"
)

var blacklistTests = []struct {
	exc []string
	srv []service
	res []service
}{
	{
		exc: nil,
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
	},
	{
		exc: []string{"?"},
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
	},
	{
		exc: []string{"host-1"},
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{{"host-2", 2000, nil}, {"host-3", 3000, nil}},
	},
	{
		exc: []string{"host-2"},
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{{"host-1", 1000, nil}, {"host-3", 3000, nil}},
	},
	{
		exc: []string{"host-3"},
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}},
	},
	{
		exc: []string{"host-1", "host-2"},
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{{"host-3", 3000, nil}},
	},
	{
		exc: []string{"host-1", "host-2", "host-3"},
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, nil}, {"host-3", 3000, nil}},
		res: []service{},
	},
}

func TestBlacklist(t *testing.T) {
	for _, test := range blacklistTests {
		t.Run("", func(t *testing.T) {
			blacklist := blacklisted(1*time.Second, serviceList(test.srv))

			for _, addr := range test.exc {
				blacklist.add(addr)
			}

			srv, err := blacklist.resolve("anything")

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(srv, test.res) {
				t.Errorf("\n%#v\n%#v", srv, test.res)
			}
		})
	}
}

func BenchmarkBlacklist(b *testing.B) {
	for _, test := range blacklistTests {
		blacklist := blacklisted(1*time.Second, serviceList(test.srv))

		for _, addr := range test.exc {
			blacklist.add(addr)
		}

		b.Run("", func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				blacklist.resolve("anything")
			}
		})
	}
}
