package foundation

import (
	"context"
	"errors"
	"testing"
)

func TestDummyFeatureFlagUsesOpenFeatureDefault(t *testing.T) {
	t.Parallel()

	evaluation := NewFeatureFlags().BootstrapProbe(context.Background())
	if evaluation.Enabled || !evaluation.Defaulted {
		t.Fatalf("noop evaluation = %+v, want visible safe default false", evaluation)
	}
}

func TestDummyFeatureFlagFailsSafe(t *testing.T) {
	t.Parallel()

	flags := newFeatureFlags(func(context.Context, string, bool) (bool, bool, error) {
		return true, false, errors.New("provider failed")
	})
	evaluation := flags.BootstrapProbe(context.Background())
	if evaluation.Enabled || !evaluation.Defaulted {
		t.Fatalf("failed evaluation = %+v, want safe default false", evaluation)
	}
}

func TestDummyFeatureFlagCanResolveTrue(t *testing.T) {
	t.Parallel()

	flags := newFeatureFlags(func(_ context.Context, key string, fallback bool) (bool, bool, error) {
		if key != BootstrapProbeFlag || fallback {
			t.Fatalf("evaluation request = (%q, %t)", key, fallback)
		}
		return true, false, nil
	})
	evaluation := flags.BootstrapProbe(context.Background())
	if !evaluation.Enabled || evaluation.Defaulted {
		t.Fatalf("resolved evaluation = %+v, want true", evaluation)
	}
}
