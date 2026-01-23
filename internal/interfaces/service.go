package interfaces

import (
	"context"

	"github.com/iwtcode/fanucService"
)

type KafkaReader interface {
	GetLastMessage(ctx context.Context, broker, topic, key string) (string, error)
}

type FanucApiService interface {
	// GetConnections возвращает список подключений с удаленного сервиса
	GetConnections(ctx context.Context, baseURL, apiKey string) ([]fanucService.MachineDTO, error)
}
