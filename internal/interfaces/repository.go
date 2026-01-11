package interfaces

import "github.com/iwtcode/fanucClient/internal/domain/entities"

type UserRepository interface {
	Save(user *entities.User) error
	GetByID(id int64) (*entities.User, error)
	UpdateState(id int64, state string) error
	UpdateBroker(id int64, broker string) error
	UpdateTopic(id int64, topic string) error
}
