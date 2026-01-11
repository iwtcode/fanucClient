package usecases

import (
	"context"
	"fmt"

	"github.com/iwtcode/fanucClient/internal/interfaces"
)

type monitoringUsecase struct {
	repo     interfaces.UserRepository
	kafkaSvc interfaces.KafkaReader
}

func NewMonitoringUsecase(repo interfaces.UserRepository, kafkaSvc interfaces.KafkaReader) interfaces.MonitoringUsecase {
	return &monitoringUsecase{
		repo:     repo,
		kafkaSvc: kafkaSvc,
	}
}

func (u *monitoringUsecase) FetchLastKafkaMessage(ctx context.Context, userID int64) (string, error) {
	user, err := u.repo.GetByID(userID)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	if user.KafkaBroker == "" || user.KafkaTopic == "" {
		return "⚠️ Please configure Kafka Broker and Topic in settings first.", nil
	}

	msg, err := u.kafkaSvc.GetLastMessage(ctx, user.KafkaBroker, user.KafkaTopic)
	if err != nil {
		return "", fmt.Errorf("kafka error: %w", err)
	}

	return msg, nil
}
