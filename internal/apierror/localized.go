package apierror

import (
	"errors"

	"connectrpc.com/connect"
	typev1 "github.com/acctbl/accountable/gen/go/accountable/type/v1"
)

func New(code connect.Code, messageKey string, cause error) *connect.Error {
	if cause == nil {
		cause = errors.New(messageKey)
	}

	err := connect.NewError(code, cause)
	if detail, detailErr := connect.NewErrorDetail(&typev1.LocalizedError{
		MessageKey: messageKey,
	}); detailErr == nil {
		err.AddDetail(detail)
	}

	return err
}
