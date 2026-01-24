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

func (u *monitoringUsecase) FetchLastKafkaMessage(ctx context.Context, targetID uint, keyID uint) (string, string, error) {
	target, err := u.repo.GetTargetByID(targetID)
	if err != nil {
		return "", "", fmt.Errorf("target not found: %w", err)
	}

	var keyString string
	// If keyID is provided (> 0), fetch the actual key string
	if keyID > 0 {
		k, err := u.repo.GetKeyByID(keyID)
		if err != nil {
			return "", "", fmt.Errorf("key not found: %w", err)
		}
		keyString = k.Key
	}

	// Use empty string for keyString if keyID == 0 (default/no key)
	foundKey, foundVal, err := u.kafkaSvc.GetLastMessage(ctx, target.Broker, target.Topic, keyString)
	if err != nil {
		return "", "", fmt.Errorf("kafka error: %w", err)
	}

	return foundKey, foundVal, nil
}
