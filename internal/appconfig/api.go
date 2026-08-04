package appconfig

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"time"

	"github.com/acctbl/accountable/internal/configfile"
	"github.com/acctbl/accountable/internal/foundation"
)

const HTTPWriteTimeout = 30 * time.Second

type API struct {
	Addr              string
	Environment       string
	ArchitectureProbe bool
	AllowedOrigins    []string
	TrustedProxies    []*net.IPNet
	TLSCertFile       string
	TLSKeyFile        string
	UnaryRPCDeadline  time.Duration
	StreamRPCDeadline time.Duration
	Foundation        foundation.Config
}

type APIFile struct {
	foundation.FileConfig
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

func LoadAPI(args []string) (API, error) {
	path, err := configfile.AbsolutePath(args)
	if err != nil {
		return API{}, err
	}
	var raw APIFile
	if err := configfile.Decode(path, &raw); err != nil {
		return API{}, err
	}
	if raw.Environment != "development" && raw.Environment != "staging" && raw.Environment != "production" {
		return API{}, errors.New("environment must be development, staging, or production")
	}
	if raw.ListenAddress == "" {
		return API{}, errors.New("listen_address is required")
	}
	if _, _, err := net.SplitHostPort(raw.ListenAddress); err != nil {
		return API{}, fmt.Errorf("listen_address: %w", err)
	}
	if raw.ArchitectureProbe == nil {
		return API{}, errors.New("architecture_probe is required")
	}
	if raw.AllowedOrigins == nil {
		return API{}, errors.New("allowed_origins is required")
	}
	if err := validateAllowedOrigins(raw.Environment, raw.AllowedOrigins); err != nil {
		return API{}, err
	}
	if raw.TrustedProxyCIDRs == nil {
		return API{}, errors.New("trusted_proxy_cidrs is required")
	}
	if raw.Environment == "production" && *raw.ArchitectureProbe {
		return API{}, errors.New("production preflight: architecture_probe must be false")
	}
	if (raw.TLSCertificate == "") != (raw.TLSPrivateKey == "") {
		return API{}, errors.New("tls_certificate_file and tls_private_key_file must be configured together")
	}
	if raw.TLSCertificate != "" && !filepath.IsAbs(raw.TLSCertificate) {
		return API{}, errors.New("tls_certificate_file must be an absolute path")
	}
	if raw.TLSPrivateKey != "" && !filepath.IsAbs(raw.TLSPrivateKey) {
		return API{}, errors.New("tls_private_key_file must be an absolute path")
	}
	if raw.TLSCertificate != "" {
		if _, err := tls.LoadX509KeyPair(raw.TLSCertificate, raw.TLSPrivateKey); err != nil {
			return API{}, errors.New("TLS identity is unavailable")
		}
	}
	unaryDeadline, err := positiveDuration("unary_rpc_timeout", raw.UnaryRPCTimeout)
	if err != nil {
		return API{}, err
	}
	streamDeadline, err := positiveDuration("stream_rpc_timeout", raw.StreamRPCTimeout)
	if err != nil {
		return API{}, err
	}
	if unaryDeadline >= HTTPWriteTimeout {
		return API{}, errors.New("unary_rpc_timeout must be shorter than the HTTP write timeout")
	}
	if streamDeadline >= HTTPWriteTimeout {
		return API{}, errors.New("stream_rpc_timeout must be shorter than the HTTP write timeout")
	}
	trusted, err := parseCIDRs(raw.TrustedProxyCIDRs)
	if err != nil {
		return API{}, fmt.Errorf("trusted_proxy_cidrs: %w", err)
	}
	foundationConfig, err := foundation.Parse(raw.Environment, path, raw.FileConfig)
	if err != nil {
		return API{}, err
	}
	return API{
		Addr: raw.ListenAddress, Environment: raw.Environment, ArchitectureProbe: *raw.ArchitectureProbe,
		AllowedOrigins: raw.AllowedOrigins, TrustedProxies: trusted,
		TLSCertFile: raw.TLSCertificate, TLSKeyFile: raw.TLSPrivateKey,
		UnaryRPCDeadline: unaryDeadline, StreamRPCDeadline: streamDeadline, Foundation: foundationConfig,
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

func positiveDuration(field, value string) (time.Duration, error) {
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
