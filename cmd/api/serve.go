package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/acctbl/accountable/gen/go/accountable/probe/v1/probev1connect"
	"github.com/acctbl/accountable/gen/go/accountable/system/v1/systemv1connect"
	"github.com/acctbl/accountable/internal/apierror"
	"github.com/acctbl/accountable/internal/appconfig"
	"github.com/acctbl/accountable/internal/bootstrap"
	"github.com/acctbl/accountable/internal/modules/probe"
	"github.com/acctbl/accountable/internal/modules/system"
	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/acctbl/accountable/internal/server"
	"github.com/google/uuid"
)

const (
	readHeaderTimeout       = 5 * time.Second
	readTimeout             = 15 * time.Second
	writeTimeout            = appconfig.HTTPWriteTimeout
	idleTimeout             = 120 * time.Second
	shutdownTimeout         = 10 * time.Second
	ordinaryMessageMaxBytes = 1 << 20
	streamMessageMaxBytes   = 256 << 10
	uploadBodyMaxBytes      = 16 << 20
	maxHeaderBytes          = 64 << 10
	maxConnections          = 128
	releaseID               = "dev"
)

type config = appconfig.API

func loadConfig(args []string) (config, error) { return appconfig.LoadAPI(args) }

type readiness struct{ ready atomic.Bool }

func (r *readiness) Set(value bool) { r.ready.Store(value) }
func (r *readiness) IsReady() bool  { return r.ready.Load() }

func newAPIHandler(config config, ready *readiness) http.Handler {
	return newAPIHandlerWithSystem(
		config,
		ready,
		system.NewSystemServer(clock.System{}),
	)
}

func newAPIHandlerWithSystem(
	config config,
	ready *readiness,
	systemServer systemv1connect.SystemServiceHandler,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /_health/live", healthHandler(func() bool { return true }))
	mux.HandleFunc("GET /_health/ready", healthHandler(ready.IsReady))

	boundary := server.BoundaryInterceptor{}
	deadline := server.DeadlineInterceptor{
		Unary:  config.UnaryRPCDeadline,
		Stream: config.StreamRPCDeadline,
	}
	availability := server.AvailabilityInterceptor{Ready: ready.IsReady}
	recoverPanic := func(ctx context.Context, _ connect.Spec, _ http.Header, _ any) error {
		return apierror.New(ctx, apierror.InternalFailure, "errors.internal", nil, 0)
	}
	ordinaryOptions := []connect.HandlerOption{
		connect.WithInterceptors(boundary, availability, deadline),
		connect.WithRecover(recoverPanic),
		connect.WithReadMaxBytes(ordinaryMessageMaxBytes),
		connect.WithSendMaxBytes(ordinaryMessageMaxBytes),
	}
	systemPath, systemHandler := systemv1connect.NewSystemServiceHandler(
		systemServer, ordinaryOptions...,
	)
	mux.Handle(systemPath, systemHandler)

	if config.ArchitectureProbe {
		probeOptions := []connect.HandlerOption{
			connect.WithInterceptors(boundary, availability, deadline),
			connect.WithRecover(recoverPanic),
			connect.WithReadMaxBytes(streamMessageMaxBytes),
			connect.WithSendMaxBytes(streamMessageMaxBytes),
		}
		probePath, probeHandler := probev1connect.NewArchitectureProbeServiceHandler(
			probe.ArchitectureProbeServer{}, probeOptions...,
		)
		mux.Handle(probePath, probeHandler)
	}

	handler := http.Handler(http.MaxBytesHandler(mux, uploadBodyMaxBytes))
	handler = trustedProxyMiddleware(config.TrustedProxies, handler)
	return credentialedCORSMiddleware(config.AllowedOrigins, handler)
}

type healthResponse struct {
	Status    string `json:"status"`
	ReleaseID string `json:"release_id"`
	RequestID string `json:"request_id"`
}

func healthHandler(healthy func() bool) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("Content-Type", "application/json")
		requestID, err := uuid.NewV7()
		if err != nil {
			requestID = uuid.Nil
		}
		response.Header().Set("X-Request-ID", requestID.String())
		status := http.StatusOK
		payload := healthResponse{Status: "ok", ReleaseID: releaseID, RequestID: requestID.String()}
		if !healthy() {
			status = http.StatusServiceUnavailable
			payload.Status = "not_ready"
		}
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func trustedProxyMiddleware(trusted []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		host, _, err := net.SplitHostPort(request.RemoteAddr)
		ip := net.ParseIP(host)
		isTrusted := err == nil && ip != nil
		if isTrusted {
			isTrusted = false
			for _, network := range trusted {
				if network.Contains(ip) {
					isTrusted = true
					break
				}
			}
		}
		if !isTrusted {
			request.Header.Del("Forwarded")
			request.Header.Del("X-Forwarded-For")
			request.Header.Del("X-Forwarded-Host")
			request.Header.Del("X-Forwarded-Proto")
		}
		next.ServeHTTP(response, request)
	})
}

func credentialedCORSMiddleware(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if _, ok := allowed[origin]; origin != "" && ok {
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Access-Control-Allow-Credentials", "true")
			response.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Trace-ID, Grpc-Status, Grpc-Message")
			response.Header().Add("Vary", "Origin")
			if request.Method == http.MethodOptions {
				response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
				response.Header().Set("Access-Control-Allow-Headers", "Accept-Language, Connect-Protocol-Version, Connect-Timeout-Ms, Content-Type, X-Trace-ID")
				response.Header().Set("Access-Control-Max-Age", "600")
				response.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(response, request)
	})
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

func serve(ctx context.Context, config config) error {
	return bootstrapAndServe(
		ctx,
		config,
		func(ctx context.Context, config bootstrap.Config) (ownedDependencySet, error) {
			return bootstrap.Build(ctx, config)
		},
		serveWithDependencies,
	)
}

type ownedDependencySet interface {
	dependencySet
	Close()
}

func bootstrapAndServe(
	ctx context.Context,
	config config,
	build func(context.Context, bootstrap.Config) (ownedDependencySet, error),
	serve func(context.Context, config, dependencySet) error,
) error {
	dependencies, err := build(ctx, config.Foundation)
	if err != nil {
		return err
	}
	defer dependencies.Close()
	return serve(ctx, config, dependencies)
}

type dependencySet interface {
	Ping(context.Context) error
}

func serveWithDependencies(ctx context.Context, config config, dependencies dependencySet) error {
	ready := &readiness{}
	srv := newHTTPServer(config.Addr, newAPIHandler(config, ready))
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}
	limited := newLimitListener(listener, maxConnections)
	ready.Set(true)
	log.Printf("listening on %s", listener.Addr())

	errCh := make(chan error, 1)
	monitorCtx, stopMonitor := context.WithCancel(ctx)
	defer stopMonitor()
	go func() {
		var err error
		if config.TLSCertFile != "" || config.TLSKeyFile != "" {
			if config.TLSCertFile == "" || config.TLSKeyFile == "" {
				err = errors.New("tls_certificate_file and tls_private_key_file must be configured together")
			} else {
				err = srv.ServeTLS(limited, config.TLSCertFile, config.TLSKeyFile)
			}
		} else {
			err = srv.Serve(limited)
		}
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()
	go monitorDependencies(monitorCtx, config.Foundation, dependencies, ready)

	select {
	case err := <-errCh:
		ready.Set(false)
		shutdownErr := shutdownServer(srv)
		if err != nil {
			return err
		}
		return shutdownErr
	case <-ctx.Done():
		ready.Set(false)
		if err := shutdownServer(srv); err != nil {
			return err
		}
		return <-errCh
	}
}

func monitorDependencies(
	ctx context.Context,
	config bootstrap.Config,
	dependencies dependencySet,
	ready *readiness,
) {
	ticker := time.NewTicker(config.ReadinessProbeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, config.CheckTimeout)
			err := dependencies.Ping(pingCtx)
			cancel()
			if ctx.Err() != nil {
				return
			}
			ready.Set(err == nil)
		}
	}
}

func shutdownServer(server *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return err
	}
	return nil
}

type limitListener struct {
	net.Listener
	sem chan struct{}
}

func newLimitListener(listener net.Listener, maximum int) *limitListener {
	return &limitListener{Listener: listener, sem: make(chan struct{}, maximum)}
}

func (l *limitListener) Accept() (net.Conn, error) {
	l.sem <- struct{}{}
	connection, err := l.Listener.Accept()
	if err != nil {
		<-l.sem
		return nil, err
	}
	return &limitedConn{Conn: connection, release: func() { <-l.sem }}, nil
}

type limitedConn struct {
	net.Conn
	release func()
	once    sync.Once
}

func (c *limitedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}
