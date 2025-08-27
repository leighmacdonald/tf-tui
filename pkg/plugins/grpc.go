package plugins

import (
	"context"
	"errors"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins/proto"
)

var ErrRPCCall = errors.New("failed to call grpc endpoint")

// GRPCClient is an implementation of KV that talks over RPC.
type GRPCClient struct {
	broker *plugin.GRPCBroker
	client proto.PlayersClient
}

func (m *GRPCClient) Get(key string) (string, error) {
	resp, err := m.client.Get(context.Background(), &proto.GetRequest{
		Key: key,
	})
	if err != nil {
		return "", errors.Join(err, ErrRPCCall)
	}

	return resp.String(), nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl Players

	broker *plugin.GRPCBroker
}

func (m *GRPCServer) Get(_ context.Context, req *proto.GetRequest) (*proto.GetResponse, error) {
	v := m.Impl.Get(req.Key)

	return &proto.GetResponse{Key: v}, nil
}

// // GRPCClient is an implementation of KV that talks over RPC.
// type GRPCAddHelperClient struct{ client proto.AddHelperClient }
//
// func (m *GRPCAddHelperClient) Sum(a, b int64) (int64, error) {
// 	resp, err := m.client.Sum(context.Background(), &proto.SumRequest{
// 		A: a,
// 		B: b,
// 	})
// 	if err != nil {
// 		hclog.Default().Info("add.Sum", "client", "start", "err", err)
// 		return 0, err
// 	}
// 	return resp.R, err
// }
//
// // Here is the gRPC server that GRPCClient talks to.
// type GRPCAddHelperServer struct {
// 	// This is the real implementation
// 	Impl AddHelper
// }

// func (m *GRPCAddHelperServer) Sum(ctx context.Context, req *proto.SumRequest) (resp *proto.SumResponse, err error) {
// 	r, err := m.Impl.Sum(req.A, req.B)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &proto.SumResponse{R: r}, err
// }
