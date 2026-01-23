package interfaces

import (
	"context"

	"github.com/iwtcode/fanucService"
)

type KafkaReader interface {
	GetLastMessage(ctx context.Context, broker, topic, key string) (string, error)
}

type FanucApiService interface {
	// Connection Management
	CreateConnection(ctx context.Context, baseURL, apiKey string, req fanucService.ConnectionRequest) (*fanucService.MachineDTO, error)
	GetConnections(ctx context.Context, baseURL, apiKey string) ([]fanucService.MachineDTO, error)
	CheckConnection(ctx context.Context, baseURL, apiKey, machineID string) (*fanucService.MachineDTO, error)
	DeleteConnection(ctx context.Context, baseURL, apiKey, machineID string) error

	// Polling Management
	StartPolling(ctx context.Context, baseURL, apiKey, machineID string, intervalMs int) error
	StopPolling(ctx context.Context, baseURL, apiKey, machineID string) error

	// Program Management
	GetControlProgram(ctx context.Context, baseURL, apiKey, machineID string) (string, error)
}
