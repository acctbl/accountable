package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/acctbl/accountable/internal/foundation"
)

type stubDependencies struct{ unavailable atomic.Bool }

func (d *stubDependencies) Check(context.Context) error {
	if d.unavailable.Load() {
		return foundation.ErrDatabaseUnavailable
	}
	return nil
}
func (d *stubDependencies) Close() {}

func TestBootstrapFailureNeverStartsServing(t *testing.T) {
	t.Parallel()

	want := errors.New("bootstrap refused")
	serveCalled := false
	err := bootstrapAndServe(
		context.Background(),
		config{},
		func(context.Context, foundation.Config) (ownedDependencySet, error) { return nil, want },
		func(context.Context, config, dependencySet) error {
			serveCalled = true
			return nil
		},
	)
	if !errors.Is(err, want) {
		t.Fatalf("bootstrapAndServe error = %v", err)
	}
	if serveCalled {
		t.Fatal("listener-serving path ran after bootstrap refusal")
	}
}

func TestRunningAPIDropsAndRecoversReadinessWithoutExiting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve API address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release API address: %v", err)
	}

	dependencies := &stubDependencies{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serveWithDependencies(ctx, config{
			Addr:              address,
			UnaryRPCDeadline:  time.Second,
			StreamRPCDeadline: time.Second,
			Foundation: foundation.Config{Database: foundation.DatabaseConfig{
				HealthCheckInterval: 5 * time.Millisecond,
				ConnectTimeout:      50 * time.Millisecond,
			}},
		}, dependencies)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("API did not stop after cancellation")
		}
	})

	assertReadinessStatus(t, address, http.StatusOK)
	dependencies.unavailable.Store(true)
	assertReadinessStatus(t, address, http.StatusServiceUnavailable)
	select {
	case err := <-done:
		t.Fatalf("API exited after a recoverable dependency failure: %v", err)
	default:
	}
	dependencies.unavailable.Store(false)
	assertReadinessStatus(t, address, http.StatusOK)
}

func assertReadinessStatus(t *testing.T, address string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + address + "/ready")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == want {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("/ready did not reach status %d", want)
}
