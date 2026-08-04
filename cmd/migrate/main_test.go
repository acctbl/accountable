package main

import (
	"context"
	"strings"
	"testing"
)

func TestRunRequiresOneAbsoluteConfigPath(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"--config", "migrate.toml"}, {"--config", "/tmp/migrate.toml", "extra"}} {
		if err := run(context.Background(), args); err == nil || !strings.Contains(err.Error(), "--config") {
			t.Fatalf("run(%q) error = %v", args, err)
		}
	}
}
