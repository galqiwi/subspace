package protocol

import (
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

const ConnectPath = "/_subspace/v1/connect"

const IdHeader = "X-Subspace-Id"

func YamuxConfig() *yamux.Config {
	config := yamux.DefaultConfig()
	config.KeepAliveInterval = 10 * time.Second
	config.ConnectionWriteTimeout = 10 * time.Second
	return config
}

func BearerToken(header http.Header) string {
	scheme, token, ok := strings.Cut(header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return token
}
