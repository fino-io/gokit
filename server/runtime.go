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
	transportDebug = "DEBUG"
	transportHTTP  = "HTTP"
	transportGRPC  = "GRPC"

	defaultReadHeaderTimeout = 5 * time.Second
	DefaultShutdownTimeout   = 10 * time.Second
)

type TransportServers struct {
	debugServer   *http.Server
	httpServer    *http.Server
	grpcServer    *grpc.Server
	debugListener net.Listener
	httpListener  net.Listener
	grpcListener  net.Listener
	grpcAddr      string
	errc          chan error
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}
}

func NewTransportServers(cfg Config, debugHandler, httpHandler http.Handler, grpcServer *grpc.Server) *TransportServers {
	return &TransportServers{
		debugServer: newHTTPServer(cfg.DebugAddr, debugHandler),
		httpServer:  newHTTPServer(cfg.HttpAddr, httpHandler),
		grpcServer:  grpcServer,
		grpcAddr:    cfg.GrpcAddr,
	}
}

func (servers *TransportServers) Start() error {
	if err := servers.listen(); err != nil {
		return err
	}

	servers.errc = make(chan error, 3)
	servers.serve()
	return nil
}

func (servers *TransportServers) listen() error {
	var err error
	if servers.debugListener, err = listen(transportDebug, servers.debugServer.Addr); err != nil {
		return err
	}
	if servers.httpListener, err = listen(transportHTTP, servers.httpServer.Addr); err != nil {
		servers.closeListeners()
		return err
	}
	if servers.grpcListener, err = listen(transportGRPC, servers.grpcAddr); err != nil {
		servers.closeListeners()
		return err
	}
	return nil
}

func (servers *TransportServers) serve() {
	go serveHTTP(servers.errc, transportDebug, servers.debugServer, servers.debugListener)
	go serveHTTP(servers.errc, transportHTTP, servers.httpServer, servers.httpListener)
	go serveGRPC(servers.errc, transportGRPC, servers.grpcServer, servers.grpcListener)
}

func (servers *TransportServers) Wait(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-servers.errc:
		return err
	}
}

func (servers *TransportServers) HTTPAddr() string {
	return listenerAddr(servers.httpListener)
}

func (servers *TransportServers) GRPCAddr() string {
	return listenerAddr(servers.grpcListener)
}

func (servers *TransportServers) closeListeners() {
	for _, listener := range []net.Listener{
		servers.debugListener,
		servers.httpListener,
		servers.grpcListener,
	} {
		if listener != nil {
			_ = listener.Close()
		}
	}
}

func (servers *TransportServers) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	var errs []error
	if servers.debugServer != nil {
		if err := servers.debugServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown %s server: %w", transportDebug, err))
		}
	}
	if servers.httpServer != nil {
		if err := servers.httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown %s server: %w", transportHTTP, err))
		}
	}
	if servers.grpcServer != nil {
		if err := shutdownGRPC(ctx, servers.grpcServer); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func listen(transport, addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s server at %q: %w", transport, addr, err)
	}
	return ln, nil
}

func serveHTTP(errc chan<- error, transport string, server *http.Server, listener net.Listener) {
	logs.Infow("begin server", "transport", transport, "address", listenerAddr(listener))

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errc <- fmt.Errorf("%s server: %w", transport, err)
	}
}

func serveGRPC(errc chan<- error, transport string, server *grpc.Server, listener net.Listener) {
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
		return fmt.Errorf("shutdown %s server: %w", transportGRPC, ctx.Err())
	}
}

func listenerAddr(listener net.Listener) string {
	if listener == nil {
		return ""
	}
	return listener.Addr().String()
}
