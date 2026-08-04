package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/acctbl/accountable/internal/apierror"
)

type AvailabilityInterceptor struct{ Ready func() bool }

func (i AvailabilityInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		if i.Ready == nil || !i.Ready() {
			return nil, apierror.New(ctx, apierror.Unavailable, "errors.unavailable", nil, 0)
		}
		return next(ctx, request)
	}
}

func (i AvailabilityInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i AvailabilityInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, connection connect.StreamingHandlerConn) error {
		if i.Ready == nil || !i.Ready() {
			return apierror.New(ctx, apierror.Unavailable, "errors.unavailable", nil, 0)
		}
		return next(ctx, connection)
	}
}
