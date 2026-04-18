package interfaces

import (
	"context"

	"github.com/iwtcode/fanucClient/internal/domain/models"
	"github.com/iwtcode/fanucService"
)

type KafkaReader interface {
	GetLastMessage(ctx context.Context, broker, topic, keyFilter string) (string, string, error)
}

type FanucApiService interface {
	CreateConnection(ctx context.Context, baseURL, apiKey string, req fanucService.ConnectionRequest) (*fanucService.MachineDTO, error)
	GetConnections(ctx context.Context, baseURL, apiKey string) ([]fanucService.MachineDTO, error)
	CheckConnection(ctx context.Context, baseURL, apiKey, machineID string) (*fanucService.MachineDTO, error)
	DeleteConnection(ctx context.Context, baseURL, apiKey, machineID string) error
	StartPolling(ctx context.Context, baseURL, apiKey, machineID string, intervalMs int) error
	StopPolling(ctx context.Context, baseURL, apiKey, machineID string) error
	GetControlProgram(ctx context.Context, baseURL, apiKey, machineID string) (string, error)
}

// NotifierService определяет методы для рассылки алертов пользователю
type NotifierService interface {
	NotifyAlarm(userID int64, payload models.AlertPayload)
}

// TelegramSender определяет метод отправки сообщения через Telegram
type TelegramSender interface {
	SendAlert(userID int64, text string) error
}

// WebSender определяет метод отправки события в браузер (SSE)
type WebSender interface {
	BroadcastToUser(userID int64, eventType string, data []byte)
}
