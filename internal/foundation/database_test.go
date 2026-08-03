package foundation

import (
	"errors"
	"testing"
)

func TestSchemaWindowRefusesUnsafeStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version int64
		dirty   bool
		want    error
	}{
		{name: "behind", version: MinimumSchemaVersion - 1, want: ErrSchemaBehind},
		{name: "minimum", version: MinimumSchemaVersion},
		{name: "maximum", version: MaximumSchemaVersion},
		{name: "unknown ahead", version: MaximumSchemaVersion + 1, want: ErrSchemaAhead},
		{name: "dirty compatible", version: MinimumSchemaVersion, dirty: true, want: ErrSchemaDirty},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSchemaState(test.version, test.dirty)
			if !errors.Is(err, test.want) {
				t.Fatalf("ValidateSchemaState(%d, %t) = %v, want %v", test.version, test.dirty, err, test.want)
			}
		})
	}
}
