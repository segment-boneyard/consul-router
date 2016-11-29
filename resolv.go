package main

// The resolver interface is implemented by diverse components involved in
// service name resolution.
type resolver interface {
	// The resolve method translates a service name into a list of service
	// endpoints that the program can use to forward requests to.
	//
	// The method should return service endpoints sorted with the best candidate
	// coming first, the program is expected to chose the first service endpoint
	// of the result list.
	//
	// If the name cannot be resolved because it could not be found the method
	// should return an empty service list, errors should be kept for runtime
	// issues that prevented the resolver from completing the request.
	resolve(name string) (srv []service, err error)
}

// The resolverFunc type implements the resolver interface and makes it possible
// for simple functions to be used as resolvers.
type resolverFunc func(string) ([]service, error)

func (f resolverFunc) resolve(name string) ([]service, error) {
	return f(name)
}

// The service structure represent an endpoint that the program uses to forward
// requests to.
type service struct {
	host string
	port int
	tags []string
}

// The serviceList type implements the resolver interface but always returns the
// same set of services, it's mostly intended to be used for tests.
type serviceList []service

func (s serviceList) resolve(name string) ([]service, error) {
	return ([]service)(s), nil
}

// The serviceMap type implements the resolver interface and provides a simple
// associative mapping between service names and service endpoints, it's mostly
// intended to be used for tests.
type serviceMap map[string][]service

func (s serviceMap) resolve(name string) ([]service, error) {
	return s[name], nil
}
