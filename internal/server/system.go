package server

import (
	"context"

	"connectrpc.com/connect"
	systemv1 "github.com/acctbl/accountable/gen/go/accountable/system/v1"
	"github.com/acctbl/accountable/internal/platform/clock"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SystemServer struct {
	Clock clock.Clock
}

func NewSystemServer(c clock.Clock) *SystemServer {
	return &SystemServer{Clock: c}
}

func (s *SystemServer) GetRuntime(
	ctx context.Context,
	req *connect.Request[systemv1.GetRuntimeRequest],
) (*connect.Response[systemv1.GetRuntimeResponse], error) {
	requestID, err := uuid.NewV7()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &systemv1.GetRuntimeResponse{
		ReleaseId:  "dev",
		ServerTime: timestamppb.New(s.Clock.Now()),
		RequestId:  requestID.String(),
	}
	return connect.NewResponse(res), nil
}
