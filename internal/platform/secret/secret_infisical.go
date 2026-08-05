package secret

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

var (
	ErrSecretSourceUnavailable = errors.New("secret source is unavailable")
	errInfisicalSecretNotFound = errors.New("infisical secret is not found")
)

type infisicalAPI interface {
	Login(context.Context) (string, error)
	Read(context.Context, string, Ref) ([]byte, error)
}

type InfisicalSecretSource struct{ api infisicalAPI }

type infisicalStoreAPI interface {
	Login(context.Context) (string, error)
	Read(context.Context, string, Ref) ([]byte, error)
	Create(context.Context, string, Ref, []byte) error
}

type InfisicalSecretStore struct{ api infisicalStoreAPI }

func NewInfisicalSecretSource(
	config Config,
	client *http.Client,
	credentials aws.CredentialsProvider,
	clock clock.Clock,
) *InfisicalSecretSource {
	return &InfisicalSecretSource{api: &infisicalHTTPAPI{
		config: config, client: client, credentials: credentials, clock: clock, signer: v4.NewSigner(),
	}}
}

func newInfisicalSecretSource(api infisicalAPI) *InfisicalSecretSource {
	return &InfisicalSecretSource{api: api}
}

func NewInfisicalSecretStore(
	config Config,
	client *http.Client,
	credentials aws.CredentialsProvider,
	clock clock.Clock,
) *InfisicalSecretStore {
	return &InfisicalSecretStore{api: &infisicalHTTPAPI{
		config: config, client: client, credentials: credentials, clock: clock, signer: v4.NewSigner(),
	}}
}

func newInfisicalSecretStore(api infisicalStoreAPI) *InfisicalSecretStore {
	return &InfisicalSecretStore{api: api}
}

func (s *InfisicalSecretStore) ResolveOrCreateBatch(
	ctx context.Context,
	refs []Ref,
	generate func(Ref) ([]byte, error),
) (map[Ref]SecretValue, error) {
	if len(refs) == 0 || generate == nil {
		return nil, ErrSecretSourceUnavailable
	}
	token, err := s.api.Login(ctx)
	if err != nil || token == "" {
		return nil, ErrSecretSourceUnavailable
	}
	values := make(map[Ref]SecretValue, len(refs))
	for _, ref := range refs {
		if _, err := ParseRef(string(ref)); err != nil {
			return nil, ErrSecretSourceUnavailable
		}
		if _, duplicate := values[ref]; duplicate {
			return nil, ErrSecretSourceUnavailable
		}
		value, err := s.api.Read(ctx, token, ref)
		if errors.Is(err, errInfisicalSecretNotFound) {
			value, err = generate(ref)
			if err == nil && len(value) > 0 && len(value) <= maximumSecretBytes {
				err = s.api.Create(ctx, token, ref, value)
			}
		}
		if err != nil || len(value) == 0 || len(value) > maximumSecretBytes {
			clear(value)
			return nil, ErrSecretSourceUnavailable
		}
		values[ref] = NewSecretValue(value)
		clear(value)
	}
	return values, nil
}

func (s *InfisicalSecretSource) ResolveBatch(ctx context.Context, refs []Ref) (map[Ref]SecretValue, error) {
	if len(refs) == 0 {
		return map[Ref]SecretValue{}, nil
	}
	seen := make(map[Ref]struct{}, len(refs))
	for _, ref := range refs {
		if _, err := ParseRef(string(ref)); err != nil {
			return nil, ErrSecretSourceUnavailable
		}
		if _, duplicate := seen[ref]; duplicate {
			return nil, ErrSecretSourceUnavailable
		}
		seen[ref] = struct{}{}
	}
	token, err := s.api.Login(ctx)
	if err != nil || token == "" {
		return nil, ErrSecretSourceUnavailable
	}
	values := make(map[Ref]SecretValue, len(refs))
	for _, ref := range refs {
		value, err := s.api.Read(ctx, token, ref)
		if err != nil || len(value) == 0 || len(value) > maximumSecretBytes {
			return nil, ErrSecretSourceUnavailable
		}
		values[ref] = NewSecretValue(value)
		clear(value)
	}
	return values, nil
}

type infisicalHTTPAPI struct {
	config      Config
	client      *http.Client
	credentials aws.CredentialsProvider
	clock       clock.Clock
	signer      *v4.Signer
}

func (a *infisicalHTTPAPI) Login(ctx context.Context) (string, error) {
	credentials, err := a.credentials.Retrieve(ctx)
	if err != nil {
		return "", err
	}
	stsBody := []byte("Action=GetCallerIdentity&Version=2011-06-15")
	stsURL := "https://sts." + a.config.AWSRegion + ".amazonaws.com/"
	stsRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, stsURL, bytes.NewReader(stsBody))
	if err != nil {
		return "", err
	}
	stsRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	payloadHash := sha256.Sum256(stsBody)
	if err := a.signer.SignHTTP(
		ctx, credentials, stsRequest, hex.EncodeToString(payloadHash[:]), "sts", a.config.AWSRegion, a.clock.Now(),
	); err != nil {
		return "", err
	}
	forwardedHeaders := make(map[string]string, len(stsRequest.Header)+2)
	for name, values := range stsRequest.Header {
		if len(values) != 1 {
			return "", ErrSecretSourceUnavailable
		}
		forwardedHeaders[name] = values[0]
	}
	forwardedHeaders["Host"] = stsRequest.Host
	forwardedHeaders["Content-Length"] = strconv.Itoa(len(stsBody))
	headerBytes, err := json.Marshal(forwardedHeaders)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(map[string]string{
		"identityId":           a.config.MachineIdentityID,
		"iamHttpRequestMethod": http.MethodPost,
		"iamRequestBody":       base64.StdEncoding.EncodeToString(stsBody),
		"iamRequestHeaders":    base64.StdEncoding.EncodeToString(headerBytes),
	})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, a.config.SiteURL+"/api/v1/auth/aws-auth/login", bytes.NewReader(payload),
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	var response struct {
		AccessToken string `json:"accessToken"`
	}
	if err := a.doJSON(request, &response); err != nil || response.AccessToken == "" {
		return "", ErrSecretSourceUnavailable
	}
	return response.AccessToken, nil
}

func (a *infisicalHTTPAPI) Read(ctx context.Context, token string, ref Ref) ([]byte, error) {
	endpoint := a.config.SiteURL + "/api/v4/secrets/" + url.PathEscape(string(ref))
	query := url.Values{
		"environment":            []string{a.config.Environment},
		"secretPath":             []string{a.config.SecretPath},
		"projectId":              []string{a.config.ProjectID},
		"type":                   []string{"shared"},
		"viewSecretValue":        []string{"true"},
		"expandSecretReferences": []string{"false"},
		"includeImports":         []string{"false"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	var response struct {
		Secret struct {
			Key   string `json:"secretKey"`
			Value string `json:"secretValue"`
		} `json:"secret"`
	}
	if err := a.doJSON(request, &response); err != nil {
		return nil, err
	}
	if response.Secret.Key != string(ref) || response.Secret.Value == "" {
		return nil, ErrSecretSourceUnavailable
	}
	return []byte(response.Secret.Value), nil
}

func (a *infisicalHTTPAPI) Create(ctx context.Context, token string, ref Ref, value []byte) error {
	endpoint := a.config.SiteURL + "/api/v4/secrets/" + url.PathEscape(string(ref))
	payload, err := json.Marshal(map[string]any{
		"environment":           a.config.Environment,
		"projectId":             a.config.ProjectID,
		"secretPath":            a.config.SecretPath,
		"secretValue":           string(value),
		"skipMultilineEncoding": true,
		"type":                  "shared",
	})
	if err != nil {
		return ErrSecretSourceUnavailable
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ErrSecretSourceUnavailable
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	var response struct {
		Secret struct {
			Key string `json:"secretKey"`
		} `json:"secret"`
	}
	if err := a.doJSON(request, &response); err != nil || response.Secret.Key != string(ref) {
		return ErrSecretSourceUnavailable
	}
	return nil
}

func (a *infisicalHTTPAPI) doJSON(request *http.Request, target any) error {
	response, err := a.client.Do(request)
	if err != nil {
		return ErrSecretSourceUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		if response.StatusCode == http.StatusNotFound {
			return errInfisicalSecretNotFound
		}
		return fmt.Errorf("%w: HTTP %s", ErrSecretSourceUnavailable, strconv.Itoa(response.StatusCode))
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maximumSecretBytes+(4<<10)))
	if err := decoder.Decode(target); err != nil {
		return ErrSecretSourceUnavailable
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return ErrSecretSourceUnavailable
	}
	return nil
}
