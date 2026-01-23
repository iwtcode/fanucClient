package usecases

import (
	"context"
	"fmt"

	"github.com/iwtcode/fanucClient/internal/interfaces"
	"github.com/iwtcode/fanucService"
)

type controlUsecase struct {
	repo   interfaces.UserRepository
	apiSvc interfaces.FanucApiService
}

func NewControlUsecase(repo interfaces.UserRepository, apiSvc interfaces.FanucApiService) interfaces.ControlUsecase {
	return &controlUsecase{
		repo:   repo,
		apiSvc: apiSvc,
	}
}

func (u *controlUsecase) getServiceConfig(svcID uint) (string, string, error) {
	svc, err := u.repo.GetServiceByID(svcID)
	if err != nil {
		return "", "", fmt.Errorf("service config not found: %w", err)
	}
	return svc.BaseURL, svc.APIKey, nil
}

func (u *controlUsecase) CreateMachine(ctx context.Context, svcID uint, endpoint, series string) (*fanucService.MachineDTO, error) {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return nil, err
	}

	req := fanucService.ConnectionRequest{
		Endpoint: endpoint,
		Series:   series,
		Timeout:  5000,
		Model:    "CreatedByBot",
	}

	return u.apiSvc.CreateConnection(ctx, baseURL, apiKey, req)
}

func (u *controlUsecase) ListMachines(ctx context.Context, svcID uint) ([]fanucService.MachineDTO, error) {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return nil, err
	}
	return u.apiSvc.GetConnections(ctx, baseURL, apiKey)
}

func (u *controlUsecase) GetMachine(ctx context.Context, svcID uint, machineID string) (*fanucService.MachineDTO, error) {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return nil, err
	}
	return u.apiSvc.CheckConnection(ctx, baseURL, apiKey, machineID)
}

func (u *controlUsecase) DeleteMachine(ctx context.Context, svcID uint, machineID string) error {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return err
	}
	return u.apiSvc.DeleteConnection(ctx, baseURL, apiKey, machineID)
}

func (u *controlUsecase) StartPolling(ctx context.Context, svcID uint, machineID string, intervalMs int) error {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return err
	}
	return u.apiSvc.StartPolling(ctx, baseURL, apiKey, machineID, intervalMs)
}

func (u *controlUsecase) StopPolling(ctx context.Context, svcID uint, machineID string) error {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return err
	}
	return u.apiSvc.StopPolling(ctx, baseURL, apiKey, machineID)
}

func (u *controlUsecase) GetProgram(ctx context.Context, svcID uint, machineID string) (string, error) {
	baseURL, apiKey, err := u.getServiceConfig(svcID)
	if err != nil {
		return "", err
	}
	return u.apiSvc.GetControlProgram(ctx, baseURL, apiKey, machineID)
}
