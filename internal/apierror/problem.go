package apierror

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	errorv1 "github.com/acctbl/accountable/gen/go/accountable/error/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
)

type Category string

const (
	InvalidInput       Category = "invalid_input"
	Unauthenticated    Category = "unauthenticated"
	Forbidden          Category = "forbidden"
	NotFound           Category = "not_found"
	AlreadyExists      Category = "already_exists"
	FailedPrecondition Category = "failed_precondition"
	StaleVersion       Category = "stale_version"
	Conflict           Category = "conflict"
	QuotaExceeded      Category = "quota_exceeded"
	RateLimited        Category = "rate_limited"
	DeadlineExceeded   Category = "deadline_exceeded"
	OutOfRange         Category = "out_of_range"
	NotImplemented     Category = "not_implemented"
	Unavailable        Category = "unavailable"
	DataLoss           Category = "data_loss"
	InternalFailure    Category = "internal_failure"
)

type FieldViolation struct {
	FieldPath string
	Code      string
}

type identityKey struct{}

type Identity struct {
	RequestID string
	TraceID   string
}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func IdentityFromContext(ctx context.Context) Identity {
	identity, _ := ctx.Value(identityKey{}).(Identity)
	return identity
}

func New(
	ctx context.Context,
	category Category,
	messageKey string,
	violations []FieldViolation,
	retryAfter time.Duration,
) *connect.Error {
	code := connectCode(category)
	problemID, uuidErr := uuid.NewV7()
	if uuidErr != nil {
		problemID = uuid.Nil
	}

	identity := IdentityFromContext(ctx)
	detail := &errorv1.ProblemDetail{
		Code:       string(category),
		MessageKey: messageKey,
		ProblemId:  problemID.String(),
		RequestId:  identity.RequestID,
	}
	for _, violation := range violations {
		detail.FieldViolations = append(detail.FieldViolations, &errorv1.FieldViolation{
			FieldPath: violation.FieldPath,
			Code:      violation.Code,
		})
	}
	if retryAfter > 0 {
		detail.RetryAfter = durationpb.New(retryAfter)
	}

	// The wire-level Connect message is deliberately generic. Client copy is
	// selected exclusively from the safe, allow-listed message_key detail.
	err := connect.NewError(code, errors.New("request failed"))
	if errorDetail, detailErr := connect.NewErrorDetail(detail); detailErr == nil {
		err.AddDetail(errorDetail)
	}
	return err
}

func EnsureSafe(ctx context.Context, err error) error {
	if err == nil || errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return New(ctx, DeadlineExceeded, "errors.deadlineExceeded", nil, 0)
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		for _, detail := range connectErr.Details() {
			value, valueErr := detail.Value()
			if valueErr == nil {
				if _, ok := value.(*errorv1.ProblemDetail); ok {
					return connectErr
				}
			}
		}
	}

	return New(ctx, InternalFailure, "errors.internal", nil, 0)
}

func connectCode(category Category) connect.Code {
	switch category {
	case InvalidInput:
		return connect.CodeInvalidArgument
	case Unauthenticated:
		return connect.CodeUnauthenticated
	case Forbidden:
		return connect.CodePermissionDenied
	case NotFound:
		return connect.CodeNotFound
	case AlreadyExists:
		return connect.CodeAlreadyExists
	case FailedPrecondition:
		return connect.CodeFailedPrecondition
	case StaleVersion, Conflict:
		return connect.CodeAborted
	case QuotaExceeded, RateLimited:
		return connect.CodeResourceExhausted
	case DeadlineExceeded:
		return connect.CodeDeadlineExceeded
	case OutOfRange:
		return connect.CodeOutOfRange
	case NotImplemented:
		return connect.CodeUnimplemented
	case Unavailable:
		return connect.CodeUnavailable
	case DataLoss:
		return connect.CodeDataLoss
	case InternalFailure:
		return connect.CodeInternal
	default:
		return connect.CodeInternal
	}
}
