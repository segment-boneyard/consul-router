package main

import "math/rand"

// The shuffled function is a resolver decorator that randomizes the list of
// services to provide a basic for of load balancing between the hosts.
func shuffled(rslv resolver) resolver {
	return resolverFunc(func(name string) (srv []service, err error) {
		if srv, err = rslv.resolve(name); err != nil {
			return
		}

		for i := range srv {
			j := rand.Intn(i + 1)
			srv[i], srv[j] = srv[j], srv[i]
		}

		return
	})
}
