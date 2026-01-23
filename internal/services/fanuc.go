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

func (s *fanucApiService) GetConnections(ctx context.Context, baseURL, apiKey string) ([]fanucService.MachineDTO, error) {
	// Инициализируем клиент SDK
	client := fanucService.NewClient(baseURL, apiKey)

	// Делаем запрос
	return client.GetConnections(ctx)
}
