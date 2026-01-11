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

func (u *settingsUsecase) SetBroker(id int64, broker string) error {
	// Сбрасываем стейт в idle после сохранения
	if err := u.repo.UpdateBroker(id, broker); err != nil {
		return err
	}
	return u.repo.UpdateState(id, entities.StateIdle)
}

func (u *settingsUsecase) SetTopic(id int64, topic string) error {
	// Сбрасываем стейт в idle после сохранения
	if err := u.repo.UpdateTopic(id, topic); err != nil {
		return err
	}
	return u.repo.UpdateState(id, entities.StateIdle)
}
