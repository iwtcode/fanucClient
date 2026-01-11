package usecases

import (
	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
)

type settingsUsecase struct {
	repo interfaces.UserRepository
}

func NewSettingsUsecase(repo interfaces.UserRepository) interfaces.SettingsUsecase {
	return &settingsUsecase{repo: repo}
}

func (u *settingsUsecase) RegisterUser(user *entities.User) error {
	return u.repo.Save(user)
}

func (u *settingsUsecase) GetUser(id int64) (*entities.User, error) {
	return u.repo.GetByID(id)
}

func (u *settingsUsecase) SetState(id int64, state string) error {
	return u.repo.UpdateState(id, state)
}

func (u *settingsUsecase) SetDraftName(id int64, name string) error {
	return u.repo.UpdateDraft(id, map[string]interface{}{
		"draft_name": name,
		"state":      entities.StateWaitingBroker,
	})
}

func (u *settingsUsecase) SetDraftBroker(id int64, broker string) error {
	return u.repo.UpdateDraft(id, map[string]interface{}{
		"draft_broker": broker,
		"state":        entities.StateWaitingTopic,
	})
}

func (u *settingsUsecase) SetDraftTopic(id int64, topic string) error {
	return u.repo.UpdateDraft(id, map[string]interface{}{
		"draft_topic": topic,
		"state":       entities.StateWaitingKey,
	})
}

func (u *settingsUsecase) SetDraftKeyAndSave(id int64, key string) error {
	// 1. Get current drafts
	user, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}

	// 2. Create Target
	target := &entities.MonitoringTarget{
		UserID: user.ID,
		Name:   user.DraftName,
		Broker: user.DraftBroker,
		Topic:  user.DraftTopic,
		Key:    key,
	}

	if err := u.repo.AddTarget(target); err != nil {
		return err
	}

	// 3. Reset State
	return u.repo.UpdateState(id, entities.StateIdle)
}

func (u *settingsUsecase) GetTargets(userID int64) ([]entities.MonitoringTarget, error) {
	return u.repo.GetTargets(userID)
}

func (u *settingsUsecase) DeleteTarget(userID int64, targetID uint) error {
	return u.repo.DeleteTarget(targetID, userID)
}

func (u *settingsUsecase) GetTargetByID(targetID uint) (*entities.MonitoringTarget, error) {
	return u.repo.GetTargetByID(targetID)
}
