package services

import (
	"context"

	"github.com/iwtcode/fanucClient/internal/interfaces"
	"github.com/iwtcode/fanucService"
)

type fanucApiService struct{}

func NewFanucApiService() interfaces.FanucApiService {
	return &fanucApiService{}
}

// Connection

func (s *fanucApiService) CreateConnection(ctx context.Context, baseURL, apiKey string, req fanucService.ConnectionRequest) (*fanucService.MachineDTO, error) {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.CreateConnection(ctx, req)
}

func (s *fanucApiService) GetConnections(ctx context.Context, baseURL, apiKey string) ([]fanucService.MachineDTO, error) {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.GetConnections(ctx)
}

func (s *fanucApiService) CheckConnection(ctx context.Context, baseURL, apiKey, machineID string) (*fanucService.MachineDTO, error) {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.CheckConnection(ctx, machineID)
}

func (s *fanucApiService) DeleteConnection(ctx context.Context, baseURL, apiKey, machineID string) error {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.DeleteConnection(ctx, machineID)
}

// Polling

func (s *fanucApiService) StartPolling(ctx context.Context, baseURL, apiKey, machineID string, intervalMs int) error {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.StartPolling(ctx, machineID, intervalMs)
}

func (s *fanucApiService) StopPolling(ctx context.Context, baseURL, apiKey, machineID string) error {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.StopPolling(ctx, machineID)
}

// Program

func (s *fanucApiService) GetControlProgram(ctx context.Context, baseURL, apiKey, machineID string) (string, error) {
	client := fanucService.NewClient(baseURL, apiKey)
	return client.GetControlProgram(ctx, machineID)
}
