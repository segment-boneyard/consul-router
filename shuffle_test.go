package main

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkShuffle(b *testing.B) {
	for _, size := range [...]int{1, 10, 100, 1000} {
		names, services := make([]string, size), make(serviceMap, size)

		for i := 0; i != size; i++ {
			name := fmt.Sprintf("host-%d", i+1)
			names[i] = name
			services[name] = []service{{name, 4242, nil}}
		}

		shuffle := shuffled(services)

		b.Run(strconv.Itoa(size), func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				shuffle.resolve(names[i%len(names)])
			}
		})
	}
}
