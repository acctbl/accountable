package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeWebTOML(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "web.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadWebRuntimeRendersTheExactPublicDocument(t *testing.T) {
	t.Parallel()

	path := writeWebTOML(t, strings.Join([]string{
		`environment = "production"`,
		`api_base_url = "https://api.accountable.example/"`,
		`architecture_probe = false`,
	}, "\n"))
	runtime, err := LoadWebRuntime(path, "2026-08-04.1")
	if err != nil {
		t.Fatalf("LoadWebRuntime: %v", err)
	}
	document, err := runtime.RuntimeConfigJSON()
	if err != nil {
		t.Fatalf("RuntimeConfigJSON: %v", err)
	}
	want := `{"schema_version":1,"api_base_url":"https://api.accountable.example",` +
		`"architecture_probe":false,"configuration_revision":"2026-08-04.1"}` + "\n"
	if string(document) != want {
		t.Fatalf("document = %s, want %s", document, want)
	}
}

func TestLoadWebRuntimeRefusesUnsafeInput(t *testing.T) {
	t.Parallel()

	valid := strings.Join([]string{
		`environment = "production"`,
		`api_base_url = "https://api.accountable.example"`,
		`architecture_probe = false`,
	}, "\n")
	for name, candidate := range map[string]struct {
		toml     string
		revision string
	}{
		"unknown field":         {valid + "\nsecret = \"x\"", "rev"},
		"missing api_base_url":  {"environment = \"production\"\narchitecture_probe = false", "rev"},
		"missing probe":         {"environment = \"production\"\napi_base_url = \"https://api.accountable.example\"", "rev"},
		"unknown environment":   {strings.Replace(valid, "production", "sandbox", 1), "rev"},
		"production probe":      {strings.Replace(valid, "architecture_probe = false", "architecture_probe = true", 1), "rev"},
		"production plain http": {strings.Replace(valid, "https://", "http://", 1), "rev"},
		"credentialed url":      {strings.Replace(valid, "https://api", "https://token@api", 1), "rev"},
		"query url":             {strings.Replace(valid, "example", "example/?x=1", 1), "rev"},
		"fragment url":          {strings.Replace(valid, "example", "example/#top", 1), "rev"},
		"non-http scheme":       {strings.Replace(valid, "https://api.accountable.example", "javascript:alert(1)", 1), "rev"},
		"empty revision":        {valid, ""},
		"illegal revision":      {valid, "rev with spaces"},
		"oversized revision":    {valid, strings.Repeat("a", 129)},
	} {
		path := writeWebTOML(t, candidate.toml)
		if _, err := LoadWebRuntime(path, candidate.revision); err == nil {
			t.Errorf("%s: LoadWebRuntime accepted unsafe input", name)
		}
	}
}
