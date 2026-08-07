package system

import (
	"context"

	"connectrpc.com/connect"
	systemv1 "github.com/acctbl/accountable/gen/go/accountable/system/v1"
	"github.com/acctbl/accountable/internal/apierror"
	"github.com/acctbl/accountable/internal/platform/clock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SystemServer struct {
	Clock     clock.Clock
	ReleaseID string
}

func NewSystemServer(c clock.Clock, releaseID string) *SystemServer {
	return &SystemServer{Clock: c, ReleaseID: releaseID}
}

func (s *SystemServer) GetRuntime(
	ctx context.Context,
	req *connect.Request[systemv1.GetRuntimeRequest],
) (*connect.Response[systemv1.GetRuntimeResponse], error) {
	identity := apierror.IdentityFromContext(ctx)
	res := &systemv1.GetRuntimeResponse{
		ReleaseId:  s.ReleaseID,
		ServerTime: timestamppb.New(s.Clock.Now()),
		RequestId:  identity.RequestID,
	}
	return connect.NewResponse(res), nil
}
