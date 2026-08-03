package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/acctbl/accountable/internal/configfile"
)

type config struct {
	Environment           string
	Addr                  string
	DistDir               string
	APIBaseURL            string
	ArchitectureProbe     bool
	ConfigurationRevision string
	TLSCertFile           string
	TLSKeyFile            string
}

type fileConfig struct {
	Environment           string `toml:"environment"`
	ListenAddress         string `toml:"listen_address"`
	DistDir               string `toml:"dist_dir"`
	APIBaseURL            string `toml:"api_base_url"`
	ArchitectureProbe     *bool  `toml:"architecture_probe"`
	ConfigurationRevision string `toml:"configuration_revision"`
	TLSCertificate        string `toml:"tls_certificate_file"`
	TLSPrivateKey         string `toml:"tls_private_key_file"`
}

type runtimeConfig struct {
	SchemaVersion         int    `json:"schema_version"`
	APIBaseURL            string `json:"api_base_url"`
	ArchitectureProbe     bool   `json:"architecture_probe"`
	ConfigurationRevision string `json:"configuration_revision"`
}

var safeRevision = regexp.MustCompile(`^[-A-Za-z0-9._]{1,128}$`)

func loadConfig(args []string) (config, error) {
	configPath, err := configfile.AbsolutePath(args)
	if err != nil {
		return config{}, err
	}
	var raw fileConfig
	if err := configfile.Decode(configPath, &raw); err != nil {
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
	if !filepath.IsAbs(raw.DistDir) {
		return config{}, errors.New("dist_dir must be an absolute path")
	}
	apiURL, err := url.Parse(raw.APIBaseURL)
	if err != nil || apiURL.Host == "" || (apiURL.Scheme != "http" && apiURL.Scheme != "https") ||
		apiURL.User != nil || apiURL.RawQuery != "" || apiURL.Fragment != "" {
		return config{}, errors.New("api_base_url must be an absolute HTTP URL without credentials, query, or fragment")
	}
	if raw.Environment == "production" && apiURL.Scheme != "https" {
		return config{}, errors.New("production api_base_url must use HTTPS")
	}
	if raw.ArchitectureProbe == nil {
		return config{}, errors.New("architecture_probe is required")
	}
	if raw.Environment == "production" && *raw.ArchitectureProbe {
		return config{}, errors.New("production preflight: architecture_probe must be false")
	}
	if !safeRevision.MatchString(raw.ConfigurationRevision) {
		return config{}, errors.New("configuration_revision is invalid")
	}
	if (raw.TLSCertificate == "") != (raw.TLSPrivateKey == "") {
		return config{}, errors.New("tls_certificate_file and tls_private_key_file must be configured together")
	}
	for field, value := range map[string]string{
		"tls_certificate_file": raw.TLSCertificate,
		"tls_private_key_file": raw.TLSPrivateKey,
	} {
		if value != "" && !filepath.IsAbs(value) {
			return config{}, fmt.Errorf("%s must be an absolute path", field)
		}
	}
	return config{
		Environment:           raw.Environment,
		Addr:                  raw.ListenAddress,
		DistDir:               filepath.Clean(raw.DistDir),
		APIBaseURL:            strings.TrimSuffix(raw.APIBaseURL, "/"),
		ArchitectureProbe:     *raw.ArchitectureProbe,
		ConfigurationRevision: raw.ConfigurationRevision,
		TLSCertFile:           raw.TLSCertificate,
		TLSKeyFile:            raw.TLSPrivateKey,
	}, nil
}

func newWebHandler(config config, assets fs.FS) (http.Handler, error) {
	if _, err := fs.Stat(assets, "index.html"); err != nil {
		return nil, errors.New("built web artifact is missing index.html")
	}

	runtime := runtimeConfig{
		SchemaVersion:         1,
		APIBaseURL:            config.APIBaseURL,
		ArchitectureProbe:     config.ArchitectureProbe,
		ConfigurationRevision: config.ConfigurationRevision,
	}
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/_runtime/config.json" {
			response.Header().Set("Cache-Control", "no-store")
			response.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(response).Encode(runtime)
			return
		}

		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			files.ServeHTTP(response, request)
			return
		}

		requestPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requestPath == "." {
			requestPath = ""
		}
		if requestPath != "" {
			if info, err := fs.Stat(assets, requestPath); err == nil && !info.IsDir() {
				if strings.HasPrefix(requestPath, "assets/") {
					response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(response, request)
				return
			}
		}

		response.Header().Set("Cache-Control", "no-cache")
		fallback := request.Clone(request.Context())
		fallback.URL.Path = "/"
		files.ServeHTTP(response, fallback)
	}), nil
}

func serve(ctx context.Context, config config) error {
	handler, err := newWebHandler(config, os.DirFS(config.DistDir))
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              config.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}
	errorsCh := make(chan error, 1)
	go func() {
		var serveErr error
		if config.TLSCertFile != "" {
			serveErr = server.ServeTLS(listener, config.TLSCertFile, config.TLSKeyFile)
		} else {
			serveErr = server.Serve(listener)
		}
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		errorsCh <- serveErr
	}()

	select {
	case err := <-errorsCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return <-errorsCh
	}
}
