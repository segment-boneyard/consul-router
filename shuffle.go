package main

import "math/rand"

// The shuffled function is a resolver decorator that randomizes the list of
// services to provide a basic for of load balancing between the hosts.
func shuffled(rslv resolver) resolver {
	return resolverFunc(func(name string) (srv []service, err error) {
		if srv, err = rslv.resolve(name); err != nil {
			return
		}

		rnd := make([]service, len(srv))

		for i, j := range rand.Perm(len(srv)) {
			rnd[i] = srv[j]
		}

		srv = rnd
		return
	})
}
