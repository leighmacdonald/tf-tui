package plugins

import (
	"context"
	"errors"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins/proto"
)

var ErrRPCCall = errors.New("failed to call grpc endpoint")

// GRPCClientPlayers is an implementation of KV that talks over RPC.
type GRPCClientLogger struct {
	broker *plugin.GRPCBroker
	client proto.LoggerClient
}

func (m *GRPCClientLogger) Info(message string) error {
	_, err := m.client.Info(context.Background(), &proto.LoggerRequest{
		Message: message,
	})
	if err != nil {
		return errors.Join(err, ErrRPCCall)
	}

	return nil
}

func (m *GRPCClientLogger) Warn(message string) error {
	_, err := m.client.Warn(context.Background(), &proto.LoggerRequest{
		Message: message,
	})
	if err != nil {
		return errors.Join(err, ErrRPCCall)
	}

	return nil
}

func (m *GRPCClientLogger) Error(message string) error {
	_, err := m.client.Error(context.Background(), &proto.LoggerRequest{
		Message: message,
	})
	if err != nil {
		return errors.Join(err, ErrRPCCall)
	}

	return nil
}

func (m *GRPCClientLogger) Debug(message string) error {
	_, err := m.client.Debug(context.Background(), &proto.LoggerRequest{
		Message: message,
	})
	if err != nil {
		return errors.Join(err, ErrRPCCall)
	}

	return nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServerLogger struct {
	// This is the real implementation
	Impl Logger

	broker *plugin.GRPCBroker
}

func (m *GRPCServerLogger) Info(_ context.Context, req *proto.LoggerRequest) (*proto.LoggerResponse, error) {
	m.Impl.Info(req.Message)

	return &proto.LoggerResponse{}, nil
}

func (m *GRPCServerLogger) Warn(_ context.Context, req *proto.LoggerRequest) (*proto.LoggerResponse, error) {
	m.Impl.Warn(req.Message)

	return &proto.LoggerResponse{}, nil
}

func (m *GRPCServerLogger) Error(_ context.Context, req *proto.LoggerRequest) (*proto.LoggerResponse, error) {
	m.Impl.Error(req.Message)

	return &proto.LoggerResponse{}, nil
}

func (m *GRPCServerLogger) Debug(_ context.Context, req *proto.LoggerRequest) (*proto.LoggerResponse, error) {
	m.Impl.Debug(req.Message)

	return &proto.LoggerResponse{}, nil
}

// GRPCClientPlayers is an implementation of KV that talks over RPC.
type GRPCClientRCON struct {
	broker *plugin.GRPCBroker
	client proto.RCONClient
}

func (m *GRPCClientRCON) Command(command string) (string, error) {
	resp, err := m.client.Command(context.Background(), &proto.RCONRequest{
		Command: command,
	})
	if err != nil {
		return "", errors.Join(err, ErrRPCCall)
	}

	return resp.String(), nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServerRCON struct {
	// This is the real implementation
	Impl RCON

	broker *plugin.GRPCBroker
}

func (m *GRPCServerRCON) Command(_ context.Context, req *proto.RCONRequest) (*proto.RCONResponse, error) {
	v := m.Impl.Command(req.Command)

	return &proto.RCONResponse{Response: v}, nil
}
