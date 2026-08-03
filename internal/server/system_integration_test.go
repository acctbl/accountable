package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"connectrpc.com/connect"
	systemv1 "github.com/acctbl/accountable/gen/go/accountable/system/v1"
	"github.com/acctbl/accountable/gen/go/accountable/system/v1/systemv1connect"
	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/acctbl/accountable/internal/server"
)

var uuidV7Pattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func TestSystemServiceGetRuntimeIntegration(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, 8, 3, 11, 22, 33, 456000000, time.UTC)

	mux := http.NewServeMux()
	path, handler := systemv1connect.NewSystemServiceHandler(
		server.NewSystemServer(clock.Fixed{Instant: want}),
	)
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := systemv1connect.NewSystemServiceClient(srv.Client(), srv.URL)
	res, err := client.GetRuntime(context.Background(), connect.NewRequest(&systemv1.GetRuntimeRequest{}))
	if err != nil {
		t.Fatalf("GetRuntime: %v", err)
	}
	if res.Msg.GetReleaseId() == "" {
		t.Fatal("expected release id")
	}
	serverTime := res.Msg.GetServerTime()
	if serverTime == nil {
		t.Fatal("expected server time")
	}
	got := serverTime.AsTime()
	if !got.Equal(want) {
		t.Fatalf("server time: got %v want %v", got, want)
	}
	if !uuidV7Pattern.MatchString(res.Msg.GetRequestId()) {
		t.Fatalf("request id %q is not a UUIDv7", res.Msg.GetRequestId())
	}
}
