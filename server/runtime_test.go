package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestTransportServersStartWaitAndShutdown(t *testing.T) {
	servers := NewTransportServers(
		Config{
			DebugAddr: "127.0.0.1:0",
			HttpAddr:  "127.0.0.1:0",
			GrpcAddr:  "127.0.0.1:0",
		},
		http.NewServeMux(),
		http.NewServeMux(),
		grpc.NewServer(),
	)

	require.NoError(t, servers.Start())
	require.NotEmpty(t, servers.HTTPAddr())
	require.NotEmpty(t, servers.GRPCAddr())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, servers.Wait(ctx))
	require.NoError(t, servers.Shutdown())
}

func TestTransportServersWaitReturnsTransportError(t *testing.T) {
	want := errors.New("transport failed")
	servers := &TransportServers{errc: make(chan error, 1)}
	servers.errc <- want
	require.ErrorIs(t, servers.Wait(context.Background()), want)
}

func TestNewDebugHandler(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(backend.Close)

	handler := NewDebugHandler(backend.Listener.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "metrics")
	}))
	debug := httptest.NewServer(handler)
	t.Cleanup(debug.Close)

	for _, path := range []string{"/livez", "/readyz", "/healthz", "/metrics", "/debug/pprof/"} {
		response, err := http.Get(debug.URL + path)
		require.NoError(t, err, path)
		require.Equal(t, http.StatusOK, response.StatusCode, path)
		require.NoError(t, response.Body.Close(), path)
	}
}
