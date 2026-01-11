package interfaces

import (
	"context"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
)

type SettingsUsecase interface {
	RegisterUser(user *entities.User) error
	GetUser(id int64) (*entities.User, error)

	// FSM Steps
	SetState(id int64, state string) error
	SetDraftName(id int64, name string) error
	SetDraftBroker(id int64, broker string) error
	SetDraftTopic(id int64, topic string) error
	SetDraftKeyAndSave(id int64, key string) error // Final step

	// Targets Management
	GetTargets(userID int64) ([]entities.MonitoringTarget, error)
	DeleteTarget(userID int64, targetID uint) error
	GetTargetByID(targetID uint) (*entities.MonitoringTarget, error)
}

type MonitoringUsecase interface {
	FetchLastKafkaMessage(ctx context.Context, targetID uint) (string, error)
}
