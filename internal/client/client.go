package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/galqiwi/subspace/internal/protocol"
	"github.com/hashicorp/yamux"
)

const minBackoff = time.Second
const maxBackoff = 30 * time.Second

var ErrUnauthorized = errors.New("server rejected client token")

type Options struct {
	ServerURL             string
	Token                 string
	Id                    string
	Port                  int
	UpstreamAuthorization string
}

func Run(ctx context.Context, options Options) error {
	backoff := minBackoff
	for {
		connected, err := runSession(ctx, options)
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(err, ErrUnauthorized) {
			return err
		}
		if connected {
			backoff = minBackoff
		}

		log.Printf("%v, reconnecting in %v", err, backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}

		if !connected {
			backoff = min(2*backoff, maxBackoff)
		}
	}
}

func runSession(ctx context.Context, options Options) (bool, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+options.Token)
	header.Set(protocol.IdHeader, options.Id)

	connectURL := strings.TrimSuffix(options.ServerURL, "/") + protocol.ConnectPath
	wsConn, resp, err := websocket.Dial(ctx, connectURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return false, ErrUnauthorized
		}
		if resp != nil && resp.StatusCode == http.StatusConflict {
			return false, fmt.Errorf("id %v is already taken", options.Id)
		}
		return false, fmt.Errorf("failed to connect: %w", err)
	}
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)

	session, err := yamux.Client(conn, protocol.YamuxConfig())
	if err != nil {
		_ = conn.Close()
		return false, fmt.Errorf("failed to start session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	go func() {
		select {
		case <-ctx.Done():
			_ = session.Close()
		case <-session.CloseChan():
		}
	}()

	httpServer := &http.Server{Handler: newUpstreamProxy(options)}
	defer func() {
		_ = httpServer.Close()
	}()

	log.Printf("connected to %v as %v", options.ServerURL, options.Id)
	err = httpServer.Serve(session)
	return true, fmt.Errorf("disconnected: %w", err)
}

func newUpstreamProxy(options Options) *httputil.ReverseProxy {
	target := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("localhost", strconv.Itoa(options.Port)),
	}
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Header.Del("Authorization")
			if options.UpstreamAuthorization != "" {
				r.Out.Header.Set("Authorization", options.UpstreamAuthorization)
			}
		},
		FlushInterval: -1,
	}
}
