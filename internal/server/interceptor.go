package server

import (
	"context"
	"regexp"

	"connectrpc.com/connect"
	"github.com/acctbl/accountable/internal/apierror"
	"github.com/google/uuid"
)

const (
	requestIDHeader = "X-Request-ID"
	traceIDHeader   = "X-Trace-ID"
)

var safeTraceID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type BoundaryInterceptor struct{}

func (BoundaryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, identity := requestIdentity(ctx, req.Header().Get(traceIDHeader))
		response, err := next(ctx, req)
		if response != nil {
			setIdentityHeaders(response.Header(), identity)
		}
		return response, apierror.EnsureSafe(ctx, err)
	}
}

func (BoundaryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (BoundaryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, identity := requestIdentity(ctx, conn.RequestHeader().Get(traceIDHeader))
		setIdentityHeaders(conn.ResponseHeader(), identity)
		return apierror.EnsureSafe(ctx, next(ctx, conn))
	}
}

func requestIdentity(ctx context.Context, requestedTraceID string) (context.Context, apierror.Identity) {
	requestID, err := uuid.NewV7()
	if err != nil {
		requestID = uuid.Nil
	}
	traceID := requestedTraceID
	if !safeTraceID.MatchString(traceID) {
		generated, generateErr := uuid.NewV7()
		if generateErr != nil {
			generated = uuid.Nil
		}
		traceID = generated.String()
	}
	identity := apierror.Identity{RequestID: requestID.String(), TraceID: traceID}
	return apierror.WithIdentity(ctx, identity), identity
}

func setIdentityHeaders(header interface{ Set(string, string) }, identity apierror.Identity) {
	header.Set(requestIDHeader, identity.RequestID)
	header.Set(traceIDHeader, identity.TraceID)
}
