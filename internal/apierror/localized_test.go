package apierror_test

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	typev1 "github.com/acctbl/accountable/gen/go/accountable/type/v1"
	"github.com/acctbl/accountable/internal/apierror"
)

func TestNewAttachesLocalizedErrorDetail(t *testing.T) {
	err := apierror.New(connect.CodeUnavailable, "errors.unknown", errors.New("dial timeout"))

	if err.Code() != connect.CodeUnavailable {
		t.Fatalf("code = %v, want %v", err.Code(), connect.CodeUnavailable)
	}

	details := err.Details()
	if len(details) != 1 {
		t.Fatalf("details = %d, want 1", len(details))
	}

	value, valueErr := details[0].Value()
	if valueErr != nil {
		t.Fatalf("detail value: %v", valueErr)
	}

	localized, ok := value.(*typev1.LocalizedError)
	if !ok {
		t.Fatalf("detail type = %T, want *typev1.LocalizedError", value)
	}
	if localized.GetMessageKey() != "errors.unknown" {
		t.Fatalf("message key = %q, want errors.unknown", localized.GetMessageKey())
	}
}
