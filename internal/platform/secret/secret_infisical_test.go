package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type fakeInfisicalAPI struct {
	token  string
	values map[Ref][]byte
	err    error
}

func (f fakeInfisicalAPI) Login(context.Context) (string, error) { return f.token, f.err }

func (f fakeInfisicalAPI) Read(_ context.Context, _ string, ref Ref) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	value, ok := f.values[ref]
	if !ok {
		return nil, errors.New("missing")
	}
	return append([]byte(nil), value...), nil
}

func TestInfisicalSecretSourceResolvesCompleteBatch(t *testing.T) {
	t.Parallel()

	source := newInfisicalSecretSource(fakeInfisicalAPI{
		token: "short-lived-token",
		values: map[Ref][]byte{
			"database/password": []byte("database-secret"),
			"crypto/key":        []byte("crypto-secret"),
		},
	})
	values, err := source.ResolveBatch(context.Background(), []Ref{"database/password", "crypto/key"})
	if err != nil {
		t.Fatalf("ResolveBatch: %v", err)
	}
	if got := string(values["database/password"].Bytes()); got != "database-secret" {
		t.Fatalf("database secret = %q", got)
	}
}

func TestInfisicalSecretSourceReturnsNoPartialBatch(t *testing.T) {
	t.Parallel()

	source := newInfisicalSecretSource(fakeInfisicalAPI{
		token:  "short-lived-token",
		values: map[Ref][]byte{"database/password": []byte("database-secret")},
	})
	values, err := source.ResolveBatch(context.Background(), []Ref{"database/password", "missing"})
	if !errors.Is(err, ErrSecretSourceUnavailable) || values != nil {
		t.Fatalf("ResolveBatch = %#v, %v; want nil, unavailable", values, err)
	}
}

func TestInfisicalSecretSourceRefusesDuplicateAndOversizedValues(t *testing.T) {
	t.Parallel()

	duplicate := newInfisicalSecretSource(fakeInfisicalAPI{token: "token"})
	if values, err := duplicate.ResolveBatch(context.Background(), []Ref{"same", "same"}); err == nil || values != nil {
		t.Fatalf("duplicate ResolveBatch = %#v, %v", values, err)
	}

	oversized := newInfisicalSecretSource(fakeInfisicalAPI{
		token:  "token",
		values: map[Ref][]byte{"large": []byte(strings.Repeat("x", maximumSecretBytes+1))},
	})
	if values, err := oversized.ResolveBatch(context.Background(), []Ref{"large"}); err == nil || values != nil {
		t.Fatalf("oversized ResolveBatch = %#v, %v", values, err)
	}
}

func TestInfisicalHTTPSourceUsesSignedAWSIdentityAndExactSecretLookup(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/auth/aws-auth/login":
			var payload struct {
				IdentityID     string `json:"identityId"`
				RequestHeaders string `json:"iamRequestHeaders"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Errorf("decode login: %v", err)
			}
			headerBytes, err := base64.StdEncoding.DecodeString(payload.RequestHeaders)
			if err != nil {
				t.Errorf("decode signed headers: %v", err)
			}
			var headers http.Header
			if err := json.Unmarshal(headerBytes, &headers); err != nil {
				t.Errorf("unmarshal signed headers: %v", err)
			}
			if payload.IdentityID != "identity-api" || headers.Get("Authorization") == "" || headers.Get("Host") == "" {
				t.Errorf("unsafe login payload: identity=%q headers=%v", payload.IdentityID, headers)
			}
			_, _ = response.Write([]byte(`{"accessToken":"short-lived"}`))
		case "/api/v4/secrets/database/password":
			if request.Header.Get("Authorization") != "Bearer short-lived" ||
				request.URL.Query().Get("projectId") != "project" ||
				request.URL.Query().Get("environment") != "production" ||
				request.URL.Query().Get("secretPath") != "/accountable/api" ||
				request.URL.Query().Get("viewSecretValue") != "true" ||
				request.URL.Query().Get("expandSecretReferences") != "false" ||
				request.URL.Query().Get("includeImports") != "false" {
				t.Errorf("unsafe secret request: %s", request.URL.String())
			}
			_, _ = response.Write([]byte(`{"secret":{"secretKey":"database/password","secretValue":"resolved"}}`))
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)

	source := NewInfisicalSecretSource(Config{
		SiteURL: server.URL, AWSRegion: "eu-west-2", ProjectID: "project", Environment: "production",
		SecretPath: "/accountable/api", MachineIdentityID: "identity-api",
	}, server.Client(), credentials.NewStaticCredentialsProvider("test-access", "test-secret", ""), clock.Fixed{
		Instant: time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC),
	})
	values, err := source.ResolveBatch(context.Background(), []Ref{"database/password"})
	if err != nil {
		t.Fatalf("ResolveBatch: %v", err)
	}
	if got := string(values["database/password"].Bytes()); got != "resolved" {
		t.Fatalf("secret = %q", got)
	}
}
