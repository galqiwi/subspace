package server

import (
	"context"
	"crypto/subtle"
	"log"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/coder/websocket"
	"github.com/galqiwi/subspace/internal/protocol"
	"github.com/hashicorp/yamux"
)

type Server struct {
	clientTokens [][]byte
	registry     *registry
}

func New(config *Config) *Server {
	var clientTokens [][]byte
	for _, token := range config.ClientTokens {
		clientTokens = append(clientTokens, []byte(token))
	}
	return &Server{
		clientTokens: clientTokens,
		registry:     newRegistry(),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == protocol.ConnectPath {
		s.handleConnect(w, r)
		return
	}
	s.handleProxy(w, r)
}

func (s *Server) isClientTokenValid(token string) bool {
	valid := false
	for _, clientToken := range s.clientTokens {
		if subtle.ConstantTimeCompare([]byte(token), clientToken) == 1 {
			valid = true
		}
	}
	return valid
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !s.isClientTokenValid(protocol.BearerToken(r.Header)) {
		http.Error(w, "invalid client token", http.StatusUnauthorized)
		return
	}

	id := r.Header.Get(protocol.IdHeader)
	if id == "" {
		http.Error(w, protocol.IdHeader+" header is not set", http.StatusBadRequest)
		return
	}

	t, ok := s.registry.claim(id)
	if !ok {
		http.Error(w, "id is already taken", http.StatusConflict)
		return
	}
	defer s.registry.release(id, t)

	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("failed to accept websocket for %v: %v", r.RemoteAddr, err)
		return
	}
	conn := websocket.NetConn(r.Context(), wsConn, websocket.MessageBinary)

	session, err := yamux.Server(conn, protocol.YamuxConfig())
	if err != nil {
		_ = conn.Close()
		log.Printf("failed to start session for %v: %v", r.RemoteAddr, err)
		return
	}
	defer func() {
		_ = session.Close()
	}()

	proxy, transport := newTunnelProxy(session)
	s.registry.activate(t, proxy, transport)
	log.Printf("client %v connected from %v", id, r.RemoteAddr)

	<-session.CloseChan()
	log.Printf("client %v disconnected", id)
}

func newTunnelProxy(session *yamux.Session) (*httputil.ReverseProxy, *http.Transport) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return session.Open()
		},
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.Out.URL.Scheme = "http"
			r.Out.URL.Host = "subspace"
			r.Out.Host = r.In.Host
		},
		Transport:     transport,
		FlushInterval: -1,
	}
	return proxy, transport
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	id := protocol.BearerToken(r.Header)
	if id == "" {
		http.Error(w, "bearer token is not set", http.StatusUnauthorized)
		return
	}

	proxy := s.registry.get(id)
	if proxy == nil {
		http.Error(w, "no client is connected with this id", http.StatusNotFound)
		return
	}
	proxy.ServeHTTP(w, r)
}
