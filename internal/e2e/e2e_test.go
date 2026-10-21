package e2e

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/galqiwi/subspace/internal/client"
	"github.com/galqiwi/subspace/internal/server"
	"github.com/stretchr/testify/require"
)

const clientToken = "test-client-token"

func startServer(t *testing.T) string {
	srv := httptest.NewServer(server.New(&server.Config{ClientTokens: []string{clientToken}}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func startBackend(t *testing.T, handler http.HandlerFunc) int {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().(*net.TCPAddr).Port
}

func echoBackend(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = fmt.Fprintf(w, "%v %v %v auth=%q body=%v", name, r.Method, r.URL.RequestURI(), r.Header.Get("Authorization"), string(body))
	}
}

type runningClient struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func (c *runningClient) stop(t *testing.T) {
	c.cancel()
	select {
	case <-c.done:
	case <-time.After(5 * time.Second):
		t.Fatal("client did not stop")
	}
}

func startClient(t *testing.T, options client.Options) *runningClient {
	ctx, cancel := context.WithCancel(context.Background())
	c := &runningClient{cancel: cancel, done: make(chan struct{})}
	go func() {
		_ = client.Run(ctx, options)
		close(c.done)
	}()
	t.Cleanup(func() { c.stop(t) })
	return c
}

func request(t *testing.T, serverURL string, id string, path string, body string) (int, string) {
	req, err := http.NewRequest(http.MethodPost, serverURL+path, strings.NewReader(body))
	require.NoError(t, err)
	if id != "" {
		req.Header.Set("Authorization", "Bearer "+id)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(respBody)
}

func waitRoutedTo(t *testing.T, serverURL string, id string, backendName string) {
	require.Eventually(t, func() bool {
		_, body := request(t, serverURL, id, "/", "")
		return strings.HasPrefix(body, backendName+" ")
	}, 10*time.Second, 20*time.Millisecond)
}

func TestRouting(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		upstreamAuthorization string
		expectedAuth          string
	}{
		{name: "stripped", upstreamAuthorization: "", expectedAuth: ""},
		{name: "replaced", upstreamAuthorization: "Bearer sk-local", expectedAuth: "Bearer sk-local"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverURL := startServer(t)
			startClient(t, client.Options{
				ServerURL:             serverURL,
				Token:                 clientToken,
				Id:                    "gpu-a",
				Port:                  startBackend(t, echoBackend("a")),
				UpstreamAuthorization: tc.upstreamAuthorization,
			})
			startClient(t, client.Options{
				ServerURL: serverURL,
				Token:     clientToken,
				Id:        "gpu-b",
				Port:      startBackend(t, echoBackend("b")),
			})
			waitRoutedTo(t, serverURL, "gpu-a", "a")
			waitRoutedTo(t, serverURL, "gpu-b", "b")

			status, body := request(t, serverURL, "gpu-a", "/v1/chat/completions?x=1", "hello")
			require.Equal(t, http.StatusOK, status)
			require.Equal(t, fmt.Sprintf("a POST /v1/chat/completions?x=1 auth=%q body=hello", tc.expectedAuth), body)
		})
	}
}

func TestRequestErrors(t *testing.T) {
	serverURL := startServer(t)

	for _, tc := range []struct {
		name           string
		id             string
		expectedStatus int
	}{
		{name: "no bearer", id: "", expectedStatus: http.StatusUnauthorized},
		{name: "unknown id", id: "gpu-unknown", expectedStatus: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, _ := request(t, serverURL, tc.id, "/", "")
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestUpstreamRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	serverURL := startServer(t)
	startClient(t, client.Options{ServerURL: serverURL, Token: clientToken, Id: "gpu", Port: port})

	require.Eventually(t, func() bool {
		status, _ := request(t, serverURL, "gpu", "/", "")
		return status == http.StatusBadGateway
	}, 10*time.Second, 20*time.Millisecond)
}

func TestInvalidClientToken(t *testing.T) {
	serverURL := startServer(t)
	err := client.Run(context.Background(), client.Options{
		ServerURL: serverURL,
		Token:     "wrong",
		Id:        "gpu",
		Port:      startBackend(t, echoBackend("a")),
	})
	require.True(t, errors.Is(err, client.ErrUnauthorized))
}

func TestDuplicateId(t *testing.T) {
	serverURL := startServer(t)
	first := startClient(t, client.Options{
		ServerURL: serverURL,
		Token:     clientToken,
		Id:        "gpu",
		Port:      startBackend(t, echoBackend("first")),
	})
	waitRoutedTo(t, serverURL, "gpu", "first")

	startClient(t, client.Options{
		ServerURL: serverURL,
		Token:     clientToken,
		Id:        "gpu",
		Port:      startBackend(t, echoBackend("second")),
	})
	time.Sleep(500 * time.Millisecond)
	_, body := request(t, serverURL, "gpu", "/", "")
	require.True(t, strings.HasPrefix(body, "first "), body)

	first.stop(t)
	waitRoutedTo(t, serverURL, "gpu", "second")
}

func TestStreaming(t *testing.T) {
	release := make(chan struct{})
	port := startBackend(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = fmt.Fprint(w, "data: second\n\n")
	})

	serverURL := startServer(t)
	startClient(t, client.Options{ServerURL: serverURL, Token: clientToken, Id: "gpu", Port: port})
	var resp *http.Response
	require.Eventually(t, func() bool {
		req, err := http.NewRequest(http.MethodGet, serverURL+"/stream", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer gpu")
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return false
		}
		return true
	}, 10*time.Second, 20*time.Millisecond)
	defer func() {
		_ = resp.Body.Close()
	}()

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "data: first\n", line)

	close(release)
	rest, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "\ndata: second\n\n", string(rest))
}
