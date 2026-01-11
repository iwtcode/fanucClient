package entities

import "time"

// State constants for FSM
const (
	StateIdle          = "idle"
	StateWaitingBroker = "waiting_broker"
	StateWaitingTopic  = "waiting_topic"
)

type User struct {
	ID        int64  `gorm:"primaryKey;autoIncrement:false"` // Telegram Chat ID
	FirstName string `gorm:"size:255"`
	UserName  string `gorm:"size:255"`

	// Kafka Configuration
	KafkaBroker string `gorm:"size:255"`
	KafkaTopic  string `gorm:"size:255"`

	// Finite State Machine
	State string `gorm:"size:50;default:'idle'"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
