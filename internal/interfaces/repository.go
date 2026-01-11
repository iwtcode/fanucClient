package interfaces

import "github.com/iwtcode/fanucClient/internal/domain/entities"

type UserRepository interface {
	Save(user *entities.User) error
	GetByID(id int64) (*entities.User, error)

	// FSM & Drafts
	UpdateState(id int64, state string) error
	UpdateDraft(id int64, updates map[string]interface{}) error

	// Targets
	AddTarget(target *entities.MonitoringTarget) error
	DeleteTarget(targetID uint, userID int64) error
	GetTargets(userID int64) ([]entities.MonitoringTarget, error)
	GetTargetByID(targetID uint) (*entities.MonitoringTarget, error)
}
