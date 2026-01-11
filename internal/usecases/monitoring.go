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

func (u *monitoringUsecase) FetchLastKafkaMessage(ctx context.Context, targetID uint) (string, error) {
	target, err := u.repo.GetTargetByID(targetID)
	if err != nil {
		return "", fmt.Errorf("target not found: %w", err)
	}

	msg, err := u.kafkaSvc.GetLastMessage(ctx, target.Broker, target.Topic, target.Key)
	if err != nil {
		return "", fmt.Errorf("kafka error: %w", err)
	}

	return msg, nil
}
