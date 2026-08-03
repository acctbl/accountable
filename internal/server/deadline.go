package server

import (
	"context"
	"time"

	"connectrpc.com/connect"
)

type DeadlineInterceptor struct {
	Unary  time.Duration
	Stream time.Duration
}

func (i DeadlineInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, i.Unary)
		defer cancel()
		return next(ctx, request)
	}
}

func (i DeadlineInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i DeadlineInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, connection connect.StreamingHandlerConn) error {
		ctx, cancel := context.WithTimeout(ctx, i.Stream)
		defer cancel()
		return next(ctx, connection)
	}
}
