package features

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
)

const BootstrapProbeFlag = "foundation.bootstrap_probe"

type FlagEvaluation struct {
	Enabled   bool
	Defaulted bool
}

type booleanEvaluator func(context.Context, string, bool) (value bool, defaulted bool, err error)

type FeatureFlags struct{ evaluate booleanEvaluator }

func NewFeatureFlags() *FeatureFlags {
	client := openfeature.NewClient("accountable")
	return newFeatureFlags(func(ctx context.Context, key string, fallback bool) (bool, bool, error) {
		details, err := client.BooleanValueDetails(ctx, key, fallback, openfeature.EvaluationContext{})
		return details.Value, details.Reason == openfeature.DefaultReason, err
	})
}

func newFeatureFlags(evaluate booleanEvaluator) *FeatureFlags {
	return &FeatureFlags{evaluate: evaluate}
}

func (f *FeatureFlags) BootstrapProbe(ctx context.Context) FlagEvaluation {
	const safeDefault = false
	value, defaulted, err := f.evaluate(ctx, BootstrapProbeFlag, safeDefault)
	if err != nil {
		return FlagEvaluation{Enabled: safeDefault, Defaulted: true}
	}
	return FlagEvaluation{Enabled: value, Defaulted: defaulted}
}
