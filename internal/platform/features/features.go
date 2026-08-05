package features

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
)

const BootstrapProbeFlag = "foundation.bootstrap_probe"

const FlagTypeBoolean = "boolean"

type FlagDeclaration struct {
	Key         string
	Type        string
	Owner       string
	Purpose     string
	SafeDefault bool
	CreatedOn   string
	ReviewBy    string
}

var flagDeclarations = map[string]FlagDeclaration{
	BootstrapProbeFlag: {
		Key: BootstrapProbeFlag, Type: FlagTypeBoolean, Owner: "Temi",
		Purpose:     "Enable the non-production foundation architecture probe",
		SafeDefault: false, CreatedOn: "2026-08-04", ReviewBy: "2026-11-04",
	},
}

func Declarations() []FlagDeclaration {
	declarations := make([]FlagDeclaration, 0, len(flagDeclarations))
	for _, declaration := range flagDeclarations {
		declarations = append(declarations, declaration)
	}
	return declarations
}

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
	safeDefault := flagDeclarations[BootstrapProbeFlag].SafeDefault
	value, defaulted, err := f.evaluate(ctx, BootstrapProbeFlag, safeDefault)
	if err != nil {
		return FlagEvaluation{Enabled: safeDefault, Defaulted: true}
	}
	return FlagEvaluation{Enabled: value, Defaulted: defaulted}
}
