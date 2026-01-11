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
	// Upsert: Создать, если нет, обновить поля, если есть, но не трогать Broker/Topic при регистрации
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"first_name", "user_name", "updated_at"}),
	}).Create(user).Error
}

func (r *userRepository) GetByID(id int64) (*entities.User, error) {
	var user entities.User
	err := r.db.First(&user, "id = ?", id).Error
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

func (r *userRepository) UpdateBroker(id int64, broker string) error {
	return r.db.Model(&entities.User{}).Where("id = ?", id).Update("kafka_broker", broker).Error
}

func (r *userRepository) UpdateTopic(id int64, topic string) error {
	return r.db.Model(&entities.User{}).Where("id = ?", id).Update("kafka_topic", topic).Error
}
