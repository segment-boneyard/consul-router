package main

import "sort"

// The preferred function is a resolver decorator that orders the service list
// where service with a matching tag will come first.
//
// An empty tag is interpreted as not filtering at all.
func preferred(tag string, rslv resolver) resolver {
	if len(tag) == 0 {
		return rslv
	}
	return resolverFunc(func(name string) (srv []service, err error) {
		if srv, err = rslv.resolve(name); err != nil {
			return
		}
		// Using a stable sort is important to preserve the previous service
		// list order among preferred and non-preferred entries.
		sort.Stable(preferredServices{tag, srv})
		return
	})
}

type preferredServices struct {
	tag string
	srv []service
}

func (s preferredServices) Len() int {
	return len(s.srv)
}

func (s preferredServices) Swap(i int, j int) {
	s.srv[i], s.srv[j] = s.srv[j], s.srv[i]
}

func (s preferredServices) Less(i int, j int) bool {
	m1 := matchPreferred(s.tag, s.srv[i])
	m2 := matchPreferred(s.tag, s.srv[j])
	return m1 != m2 && m1
}

func matchPreferred(tag string, srv service) bool {
	for _, t := range srv.tags {
		if t == tag {
			return true
		}
	}
	return false
}
