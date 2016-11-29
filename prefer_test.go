package main

import (
	"reflect"
	"testing"
)

var preferredTests = []struct {
	tag string
	srv []service
	res []service
}{
	{
		tag: "",
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}},
		res: []service{{"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}},
	},
	{
		tag: "A",
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}},
		res: []service{{"host-3", 3000, []string{"A", "C"}}, {"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}},
	},
	{
		tag: "B",
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}},
		res: []service{{"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}},
	},
	{
		tag: "C",
		srv: []service{{"host-1", 1000, nil}, {"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}},
		res: []service{{"host-2", 2000, []string{"C"}}, {"host-3", 3000, []string{"A", "C"}}, {"host-1", 1000, nil}},
	},
}

func TestPrefer(t *testing.T) {
	for _, test := range preferredTests {
		t.Run(test.tag, func(t *testing.T) {
			prefer := preferred(test.tag, serviceList(test.srv))

			srv, err := prefer.resolve("anything")

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(srv, test.res) {
				t.Errorf("\n%#v\n%#v", srv, test.res)
			}
		})
	}
}

func BenchmarkPrefer(b *testing.B) {
	for _, test := range preferredTests {
		prefer := preferred(test.tag, serviceList(test.srv))

		b.Run(test.tag, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				prefer.resolve("anything")
			}
		})
	}
}
