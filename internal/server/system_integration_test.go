package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	systemv1 "github.com/acctbl/accountable/gen/go/accountable/system/v1"
	"github.com/acctbl/accountable/gen/go/accountable/system/v1/systemv1connect"
	"github.com/acctbl/accountable/internal/server"
)

func TestSystemServiceGetRuntimeIntegration(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	path, handler := systemv1connect.NewSystemServiceHandler(&server.SystemServer{})
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
	if res.Msg.GetServerTime() == nil {
		t.Fatal("expected server time")
	}
}
