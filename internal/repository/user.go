package repository

import (
	"errors"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) interfaces.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Save(user *entities.User) error {
	// Upsert: Создать, если нет, обновить поля
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"first_name", "user_name", "updated_at"}),
	}).Create(user).Error
}

func (r *userRepository) GetByID(id int64) (*entities.User, error) {
	var user entities.User
	err := r.db.Preload("Targets").First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) UpdateState(id int64, state string) error {
	return r.db.Model(&entities.User{}).Where("id = ?", id).Update("state", state).Error
}

func (r *userRepository) UpdateDraft(id int64, updates map[string]interface{}) error {
	return r.db.Model(&entities.User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *userRepository) AddTarget(target *entities.MonitoringTarget) error {
	return r.db.Create(target).Error
}

func (r *userRepository) DeleteTarget(targetID uint, userID int64) error {
	return r.db.Delete(&entities.MonitoringTarget{}, "id = ? AND user_id = ?", targetID, userID).Error
}

func (r *userRepository) GetTargets(userID int64) ([]entities.MonitoringTarget, error) {
	var targets []entities.MonitoringTarget
	err := r.db.Where("user_id = ?", userID).Find(&targets).Error
	return targets, err
}

func (r *userRepository) GetTargetByID(targetID uint) (*entities.MonitoringTarget, error) {
	var t entities.MonitoringTarget
	err := r.db.First(&t, "id = ?", targetID).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}
