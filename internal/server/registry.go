package server

import (
	"net/http"
	"net/http/httputil"
	"sync"
)

type tunnel struct {
	proxy     *httputil.ReverseProxy
	transport *http.Transport
}

type registry struct {
	mu      sync.Mutex
	tunnels map[string]*tunnel
}

func newRegistry() *registry {
	return &registry{tunnels: make(map[string]*tunnel)}
}

func (r *registry) claim(id string) (*tunnel, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tunnels[id]; ok {
		return nil, false
	}
	t := &tunnel{}
	r.tunnels[id] = t
	return t, true
}

func (r *registry) activate(t *tunnel, proxy *httputil.ReverseProxy, transport *http.Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t.proxy = proxy
	t.transport = transport
}

func (r *registry) release(id string, t *tunnel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tunnels[id] != t {
		return
	}
	delete(r.tunnels, id)
	if t.transport != nil {
		t.transport.CloseIdleConnections()
	}
}

func (r *registry) get(id string) *httputil.ReverseProxy {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tunnels[id]
	if !ok {
		return nil
	}
	return t.proxy
}
