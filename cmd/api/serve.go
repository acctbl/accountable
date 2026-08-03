package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/acctbl/accountable/gen/go/accountable/probe/v1/probev1connect"
	"github.com/acctbl/accountable/gen/go/accountable/system/v1/systemv1connect"
	"github.com/acctbl/accountable/internal/apierror"
	"github.com/acctbl/accountable/internal/configfile"
	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/acctbl/accountable/internal/server"
)

const (
	readHeaderTimeout       = 5 * time.Second
	readTimeout             = 15 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 120 * time.Second
	shutdownTimeout         = 10 * time.Second
	ordinaryMessageMaxBytes = 1 << 20
	streamMessageMaxBytes   = 256 << 10
	uploadBodyMaxBytes      = 16 << 20
	maxHeaderBytes          = 64 << 10
	maxConnections          = 128
)

type config struct {
	Addr              string
	Environment       string
	ArchitectureProbe bool
	AllowedOrigins    []string
	TrustedProxies    []*net.IPNet
	TLSCertFile       string
	TLSKeyFile        string
	UnaryRPCDeadline  time.Duration
	StreamRPCDeadline time.Duration
}

type fileConfig struct {
	Environment       string   `toml:"environment"`
	ListenAddress     string   `toml:"listen_address"`
	ArchitectureProbe *bool    `toml:"architecture_probe"`
	AllowedOrigins    []string `toml:"allowed_origins"`
	TrustedProxyCIDRs []string `toml:"trusted_proxy_cidrs"`
	TLSCertificate    string   `toml:"tls_certificate_file"`
	TLSPrivateKey     string   `toml:"tls_private_key_file"`
	UnaryRPCTimeout   string   `toml:"unary_rpc_timeout"`
	StreamRPCTimeout  string   `toml:"stream_rpc_timeout"`
}

func loadConfig(args []string) (config, error) {
	path, err := configfile.AbsolutePath(args)
	if err != nil {
		return config{}, err
	}

	var raw fileConfig
	if err := configfile.Decode(path, &raw); err != nil {
		return config{}, err
	}
	if raw.Environment != "development" && raw.Environment != "staging" && raw.Environment != "production" {
		return config{}, errors.New("environment must be development, staging, or production")
	}
	if raw.ListenAddress == "" {
		return config{}, errors.New("listen_address is required")
	}
	if _, _, err := net.SplitHostPort(raw.ListenAddress); err != nil {
		return config{}, fmt.Errorf("listen_address: %w", err)
	}
	if raw.ArchitectureProbe == nil {
		return config{}, errors.New("architecture_probe is required")
	}
	if raw.AllowedOrigins == nil {
		return config{}, errors.New("allowed_origins is required")
	}
	if err := validateAllowedOrigins(raw.Environment, raw.AllowedOrigins); err != nil {
		return config{}, err
	}
	if raw.TrustedProxyCIDRs == nil {
		return config{}, errors.New("trusted_proxy_cidrs is required")
	}
	if raw.Environment == "production" && *raw.ArchitectureProbe {
		return config{}, errors.New("production preflight: architecture_probe must be false")
	}
	if (raw.TLSCertificate == "") != (raw.TLSPrivateKey == "") {
		return config{}, errors.New("tls_certificate_file and tls_private_key_file must be configured together")
	}
	if raw.TLSCertificate != "" && !filepath.IsAbs(raw.TLSCertificate) {
		return config{}, errors.New("tls_certificate_file must be an absolute path")
	}
	if raw.TLSPrivateKey != "" && !filepath.IsAbs(raw.TLSPrivateKey) {
		return config{}, errors.New("tls_private_key_file must be an absolute path")
	}
	unaryDeadline, err := parsePositiveDuration("unary_rpc_timeout", raw.UnaryRPCTimeout)
	if err != nil {
		return config{}, err
	}
	streamDeadline, err := parsePositiveDuration("stream_rpc_timeout", raw.StreamRPCTimeout)
	if err != nil {
		return config{}, err
	}
	if unaryDeadline >= writeTimeout {
		return config{}, errors.New("unary_rpc_timeout must be shorter than the HTTP write timeout")
	}
	if streamDeadline >= writeTimeout {
		return config{}, errors.New("stream_rpc_timeout must be shorter than the HTTP write timeout")
	}
	trusted, err := parseCIDRs(raw.TrustedProxyCIDRs)
	if err != nil {
		return config{}, fmt.Errorf("trusted_proxy_cidrs: %w", err)
	}
	return config{
		Addr:              raw.ListenAddress,
		Environment:       raw.Environment,
		ArchitectureProbe: *raw.ArchitectureProbe,
		AllowedOrigins:    raw.AllowedOrigins,
		TrustedProxies:    trusted,
		TLSCertFile:       raw.TLSCertificate,
		TLSKeyFile:        raw.TLSPrivateKey,
		UnaryRPCDeadline:  unaryDeadline,
		StreamRPCDeadline: streamDeadline,
	}, nil
}

func validateAllowedOrigins(environment string, values []string) error {
	for _, value := range values {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("allowed_origins must contain absolute HTTP origins without credentials, paths, queries, or fragments")
		}
		if environment == "production" && parsed.Scheme != "https" {
			return errors.New("production allowed_origins must use HTTPS")
		}
	}
	return nil
}

func parsePositiveDuration(field string, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", field)
	}
	return duration, nil
}

func parseCIDRs(values []string) ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}
	return networks, nil
}

type readiness struct{ ready atomic.Bool }

func (r *readiness) Set(value bool) { r.ready.Store(value) }
func (r *readiness) IsReady() bool  { return r.ready.Load() }

func newAPIHandler(config config, ready *readiness) http.Handler {
	return newAPIHandlerWithSystem(
		config,
		ready,
		server.NewSystemServer(clock.System{}),
	)
}

func newAPIHandlerWithSystem(
	config config,
	ready *readiness,
	systemServer systemv1connect.SystemServiceHandler,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /live", healthHandler(func() bool { return true }))
	mux.HandleFunc("GET /ready", healthHandler(ready.IsReady))

	boundary := server.BoundaryInterceptor{}
	deadline := server.DeadlineInterceptor{
		Unary:  config.UnaryRPCDeadline,
		Stream: config.StreamRPCDeadline,
	}
	recoverPanic := func(ctx context.Context, _ connect.Spec, _ http.Header, _ any) error {
		return apierror.New(ctx, apierror.InternalFailure, "errors.internal", nil, 0)
	}
	ordinaryOptions := []connect.HandlerOption{
		connect.WithInterceptors(boundary, deadline),
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
			connect.WithInterceptors(boundary, deadline),
			connect.WithRecover(recoverPanic),
			connect.WithReadMaxBytes(streamMessageMaxBytes),
			connect.WithSendMaxBytes(streamMessageMaxBytes),
		}
		probePath, probeHandler := probev1connect.NewArchitectureProbeServiceHandler(
			server.ArchitectureProbeServer{}, probeOptions...,
		)
		mux.Handle(probePath, probeHandler)
	}

	handler := http.Handler(http.MaxBytesHandler(mux, uploadBodyMaxBytes))
	handler = trustedProxyMiddleware(config.TrustedProxies, handler)
	handler = drainingMiddleware(ready, handler)
	return credentialedCORSMiddleware(config.AllowedOrigins, handler)
}

func healthHandler(healthy func() bool) http.HandlerFunc {
	return func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		payload := map[string]string{"status": "ok"}
		if !healthy() {
			status = http.StatusServiceUnavailable
			payload["status"] = "not_ready"
		}
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(payload)
	}
}

func drainingMiddleware(ready *readiness, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/live" && request.URL.Path != "/ready" && !ready.IsReady() {
			http.Error(response, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(response, request)
	})
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
	ready := &readiness{}
	srv := newHTTPServer(config.Addr, newAPIHandler(config, ready))
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}
	limited := newLimitListener(listener, maxConnections)
	ready.Set(true)

	errCh := make(chan error, 1)
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

	select {
	case err := <-errCh:
		ready.Set(false)
		return err
	case <-ctx.Done():
		ready.Set(false)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return err
		}
		return <-errCh
	}
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
