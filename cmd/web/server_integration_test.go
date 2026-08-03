package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRunRequiresOneAbsoluteConfigPath(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		nil,
		{"--config", "web.toml"},
		{"--config", "/tmp/web.toml", "unexpected"},
	} {
		err := run(context.Background(), args)
		if err == nil || !strings.Contains(err.Error(), "--config") {
			t.Fatalf("run(%q) error = %v, want --config usage error", args, err)
		}
	}
}

func TestProductionHandlerServesRuntimeConfigAndBuiltArtifact(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<!doctype html><div id=\"root\"></div>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("globalThis.booted = true")},
	}
	handler, err := newWebHandler(config{
		APIBaseURL:            "https://api.example",
		ArchitectureProbe:     false,
		ConfigurationRevision: "release-42",
	}, fs.FS(assets))
	if err != nil {
		t.Fatalf("new web handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/_runtime/config.json")
	if err != nil {
		t.Fatalf("runtime config: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close runtime config response: %v", err)
		}
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("runtime config status = %d", response.StatusCode)
	}
	if cache := response.Header.Get("Cache-Control"); cache != "no-store" {
		t.Fatalf("runtime config Cache-Control = %q", cache)
	}
	var runtime runtimeConfig
	if err := json.NewDecoder(response.Body).Decode(&runtime); err != nil {
		t.Fatalf("decode runtime config: %v", err)
	}
	if runtime.SchemaVersion != 1 ||
		runtime.APIBaseURL != "https://api.example" ||
		runtime.ArchitectureProbe ||
		runtime.ConfigurationRevision != "release-42" {
		t.Fatalf("runtime config = %+v", runtime)
	}

	index, err := server.Client().Get(server.URL + "/")
	if err != nil {
		t.Fatalf("built index: %v", err)
	}
	t.Cleanup(func() {
		if err := index.Body.Close(); err != nil {
			t.Errorf("close built index response: %v", err)
		}
	})
	if index.StatusCode != http.StatusOK {
		t.Fatalf("built index status = %d", index.StatusCode)
	}
}
