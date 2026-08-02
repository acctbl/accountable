package server

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	systemv1 "github.com/acctbl/accountable/gen/go/accountable/system/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SystemServer struct{}

func (s *SystemServer) GetRuntime(
	ctx context.Context,
	req *connect.Request[systemv1.GetRuntimeRequest],
) (*connect.Response[systemv1.GetRuntimeResponse], error) {
	res := &systemv1.GetRuntimeResponse{
		ReleaseId:  "dev",
		ServerTime: timestamppb.Now(),
		RequestId:  fmt.Sprintf("req-%d", time.Now().UnixNano()),
	}
	return connect.NewResponse(res), nil
}
