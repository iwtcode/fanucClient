package interfaces

import (
	"context"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
)

type SettingsUsecase interface {
	RegisterUser(user *entities.User) error
	GetUser(id int64) (*entities.User, error)
	SetState(id int64, state string) error
	SetBroker(id int64, broker string) error
	SetTopic(id int64, topic string) error
}

type MonitoringUsecase interface {
	FetchLastKafkaMessage(ctx context.Context, userID int64) (string, error)
}
