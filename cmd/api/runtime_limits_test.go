package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTrustedProxyMiddlewareStripsUntrustedForwardingHeaders(t *testing.T) {
	t.Parallel()
	_, trustedNetwork, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("parse trusted network: %v", err)
	}
	seen := make(chan string, 2)
	handler := trustedProxyMiddleware([]*net.IPNet{trustedNetwork}, http.HandlerFunc(
		func(_ http.ResponseWriter, request *http.Request) {
			seen <- request.Header.Get("X-Forwarded-For")
		},
	))

	untrusted := httptest.NewRequest(http.MethodGet, "http://api.example/live", nil)
	untrusted.RemoteAddr = "192.0.2.10:1234"
	untrusted.Header.Set("X-Forwarded-For", "203.0.113.2")
	handler.ServeHTTP(httptest.NewRecorder(), untrusted)
	if forwarded := <-seen; forwarded != "" {
		t.Fatalf("untrusted forwarding header retained: %q", forwarded)
	}

	trusted := httptest.NewRequest(http.MethodGet, "http://api.example/live", nil)
	trusted.RemoteAddr = "10.1.2.3:1234"
	trusted.Header.Set("X-Forwarded-For", "203.0.113.2")
	handler.ServeHTTP(httptest.NewRecorder(), trusted)
	if forwarded := <-seen; forwarded != "203.0.113.2" {
		t.Fatalf("trusted forwarding header = %q", forwarded)
	}
}

func TestHTTPServerHasBoundedWriteTimeout(t *testing.T) {
	t.Parallel()

	server := newHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if server.WriteTimeout <= 0 {
		t.Fatalf("WriteTimeout = %v, want a positive bound", server.WriteTimeout)
	}
}

func TestLimitListenerReleasesCapacityWhenConnectionCloses(t *testing.T) {
	t.Parallel()
	listener := &queuedListener{connections: make(chan net.Conn, 2)}
	firstServer, firstClient := net.Pipe()
	secondServer, secondClient := net.Pipe()
	t.Cleanup(func() {
		_ = firstClient.Close()
		_ = secondClient.Close()
		_ = secondServer.Close()
	})
	listener.connections <- firstServer
	listener.connections <- secondServer
	limited := newLimitListener(listener, 1)

	first, err := limited.Accept()
	if err != nil {
		t.Fatalf("accept first: %v", err)
	}
	secondAccepted := make(chan net.Conn, 1)
	go func() {
		connection, acceptErr := limited.Accept()
		if acceptErr == nil {
			secondAccepted <- connection
		}
	}()
	select {
	case <-secondAccepted:
		t.Fatal("second connection bypassed limit")
	case <-time.After(20 * time.Millisecond):
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}
	select {
	case second := <-secondAccepted:
		_ = second.Close()
	case <-time.After(time.Second):
		t.Fatal("closing first connection did not release capacity")
	}
}

type queuedListener struct{ connections chan net.Conn }

func (l *queuedListener) Accept() (net.Conn, error) { return <-l.connections, nil }
func (l *queuedListener) Close() error              { return nil }
func (l *queuedListener) Addr() net.Addr            { return testAddr("queued") }

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

var _ io.Closer = (*limitedConn)(nil)
