package appconfig

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/acctbl/accountable/internal/configfile"
)

const runtimeConfigSchemaVersion = 1

var configurationRevisionPattern = regexp.MustCompile(`^[-A-Za-z0-9._]{1,128}$`)

type WebRuntime struct {
	Environment           string
	APIBaseURL            string
	ArchitectureProbe     bool
	ConfigurationRevision string
}

type WebRuntimeFile struct {
	Environment       string `toml:"environment"`
	APIBaseURL        string `toml:"api_base_url"`
	ArchitectureProbe *bool  `toml:"architecture_probe"`
}

func LoadWebRuntime(path, revision string) (WebRuntime, error) {
	var raw WebRuntimeFile
	if err := configfile.Decode(path, &raw); err != nil {
		return WebRuntime{}, err
	}
	if raw.Environment != "development" && raw.Environment != "staging" && raw.Environment != "production" {
		return WebRuntime{}, errors.New("environment must be development, staging, or production")
	}
	if raw.ArchitectureProbe == nil {
		return WebRuntime{}, errors.New("architecture_probe is required")
	}
	if raw.Environment == "production" && *raw.ArchitectureProbe {
		return WebRuntime{}, errors.New("production web runtime: architecture_probe must be false")
	}
	baseURL, err := validateAPIBaseURL(raw.Environment, raw.APIBaseURL)
	if err != nil {
		return WebRuntime{}, err
	}
	if !configurationRevisionPattern.MatchString(revision) {
		return WebRuntime{}, errors.New("revision must be 1 to 128 characters using only letters, digits, '.', '_', or '-'")
	}
	return WebRuntime{
		Environment: raw.Environment, APIBaseURL: baseURL,
		ArchitectureProbe: *raw.ArchitectureProbe, ConfigurationRevision: revision,
	}, nil
}

func validateAPIBaseURL(environment, value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("api_base_url must be an absolute HTTP URL without credentials, queries, or fragments")
	}
	if environment == "production" && parsed.Scheme != "https" {
		return "", errors.New("production web runtime: api_base_url must use HTTPS")
	}
	return strings.TrimSuffix(value, "/"), nil
}

func (w WebRuntime) RuntimeConfigJSON() ([]byte, error) {
	document, err := json.Marshal(struct {
		SchemaVersion         int    `json:"schema_version"`
		APIBaseURL            string `json:"api_base_url"`
		ArchitectureProbe     bool   `json:"architecture_probe"`
		ConfigurationRevision string `json:"configuration_revision"`
	}{
		SchemaVersion:         runtimeConfigSchemaVersion,
		APIBaseURL:            w.APIBaseURL,
		ArchitectureProbe:     w.ArchitectureProbe,
		ConfigurationRevision: w.ConfigurationRevision,
	})
	if err != nil {
		return nil, err
	}
	return append(document, '\n'), nil
}
