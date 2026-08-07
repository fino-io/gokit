package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fino-io/finokit/logs"
	"google.golang.org/grpc"
)

const (
	TransportDEBUG = "DEBUG"
	TransportHTTP  = "HTTP"
	TransportGRPC  = "GRPC"

	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultShutdownTimeout   = 10 * time.Second
)

type TransportServers struct {
	DebugServer     *http.Server
	HTTPServer      *http.Server
	GRPCServer      *grpc.Server
	DebugListener   net.Listener
	HTTPListener    net.Listener
	GRPCListener    net.Listener
	ShutdownTimeout time.Duration
}

func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
	}
}

func (servers *TransportServers) Listen(debugAddr, httpAddr, grpcAddr string) error {
	var err error
	if servers.DebugListener, err = Listen(TransportDEBUG, debugAddr); err != nil {
		return err
	}
	if servers.HTTPListener, err = Listen(TransportHTTP, httpAddr); err != nil {
		return err
	}
	if servers.GRPCListener, err = Listen(TransportGRPC, grpcAddr); err != nil {
		return err
	}
	return nil
}

func (servers *TransportServers) Serve(errc chan<- error) {
	go ServeHTTP(errc, TransportDEBUG, servers.DebugServer, servers.DebugListener)
	go ServeHTTP(errc, TransportHTTP, servers.HTTPServer, servers.HTTPListener)
	go ServeGRPC(errc, TransportGRPC, servers.GRPCServer, servers.GRPCListener)
}

func (servers *TransportServers) HTTPAddr() string {
	return listenerAddr(servers.HTTPListener)
}

func (servers *TransportServers) GRPCAddr() string {
	return listenerAddr(servers.GRPCListener)
}

func (servers *TransportServers) CloseListeners() {
	for _, listener := range []net.Listener{
		servers.DebugListener,
		servers.HTTPListener,
		servers.GRPCListener,
	} {
		if listener != nil {
			_ = listener.Close()
		}
	}
}

func (servers *TransportServers) Shutdown() error {
	timeout := servers.ShutdownTimeout
	if timeout <= 0 {
		timeout = DefaultShutdownTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var errs []error
	if servers.DebugServer != nil {
		if err := servers.DebugServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown %s server: %w", TransportDEBUG, err))
		}
	}
	if servers.HTTPServer != nil {
		if err := servers.HTTPServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown %s server: %w", TransportHTTP, err))
		}
	}
	if servers.GRPCServer != nil {
		if err := shutdownGRPC(ctx, servers.GRPCServer); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func Listen(transport, addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s server at %q: %w", transport, addr, err)
	}
	return ln, nil
}

func ServeHTTP(errc chan<- error, transport string, server *http.Server, listener net.Listener) {
	logs.Infow("begin server", "transport", transport, "address", listenerAddr(listener))

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errc <- fmt.Errorf("%s server: %w", transport, err)
	}
}

func ServeGRPC(errc chan<- error, transport string, server *grpc.Server, listener net.Listener) {
	logs.Infow("begin server", "transport", transport, "address", listenerAddr(listener))

	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		errc <- fmt.Errorf("%s server: %w", transport, err)
	}
}

func shutdownGRPC(ctx context.Context, server *grpc.Server) error {
	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		server.Stop()
		return fmt.Errorf("shutdown %s server: %w", TransportGRPC, ctx.Err())
	}
}

func listenerAddr(listener net.Listener) string {
	if listener == nil {
		return ""
	}
	return listener.Addr().String()
}
