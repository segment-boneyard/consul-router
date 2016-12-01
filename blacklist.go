package main

import (
	"runtime"
	"sync"
	"time"
)

// The blacklist type is a data structure that is similar to a cache, it keeps
// values around for a configurable amount of time. It is used to implement
// blacklisting of hosts that have had connection failures.
type blacklist struct {
	// Immutable fields of the blacklist data structure.
	timeout time.Duration
	rslv    resolver
	done    chan struct{}

	// Mutable fields of the blacklist data structure, the mutex must be locked
	// to access them concurrently.
	mutex sync.RWMutex
	addr  map[string]time.Time
}

func blacklisted(timeout time.Duration, rslv resolver) *blacklist {
	b := &blacklist{
		timeout: timeout,
		rslv:    rslv,
		done:    make(chan struct{}),
		addr:    make(map[string]time.Time),
	}
	runtime.SetFinalizer(b, func(b *blacklist) { close(b.done) })
	go blacklistVacuum(&b.mutex, b.addr, b.done)
	return b
}

func (b *blacklist) add(addr string) {
	now := time.Now()
	lim := now.Add(b.timeout)

	b.mutex.Lock()

	// Checking for existence so the address expiration time doesn't get
	// updated after it was set.
	if exp, exist := b.addr[addr]; !exist || now.After(exp) {
		b.addr[addr] = lim
	}

	b.mutex.Unlock()
}

func (b *blacklist) resolve(name string) (srv []service, err error) {
	if srv, err = b.rslv.resolve(name); err != nil {
		return
	}

	i := 0
	now := time.Now()
	b.mutex.RLock()

	for _, s := range srv { // filter out black-listed hosts
		if exp, bad := b.addr[s.host]; !bad || now.After(exp) {
			srv[i] = s
			i++
		}
	}

	b.mutex.RUnlock()
	srv = srv[:i]
	return
}

func blacklistVacuum(mutex *sync.RWMutex, blacklist map[string]time.Time, done <-chan struct{}) {
	const max = 100

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case now := <-ticker.C:
			blacklistVacuumPass(mutex, blacklist, now, max)
		}
	}
}

func blacklistVacuumPass(mutex *sync.RWMutex, blacklist map[string]time.Time, now time.Time, max int) {
	i := 0
	mutex.Lock()

	for addr, exp := range blacklist {
		if i++; i > max {
			break
		}
		if now.After(exp) {
			delete(blacklist, addr)
		}
	}

	mutex.Unlock()
}
