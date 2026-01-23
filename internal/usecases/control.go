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

func (u *controlUsecase) ListMachines(ctx context.Context, svcID uint) ([]fanucService.MachineDTO, error) {
	// 1. Получаем конфиг сервиса из БД
	svc, err := u.repo.GetServiceByID(svcID)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// 2. Делаем запрос через API
	machines, err := u.apiSvc.GetConnections(ctx, svc.BaseURL, svc.APIKey)
	if err != nil {
		return nil, fmt.Errorf("api error: %w", err)
	}

	return machines, nil
}
