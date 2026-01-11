package entities

import "time"

// State constants for FSM
const (
	StateIdle          = "idle"
	StateWaitingName   = "waiting_name"
	StateWaitingBroker = "waiting_broker"
	StateWaitingTopic  = "waiting_topic"
	StateWaitingKey    = "waiting_key"
)

type User struct {
	ID        int64  `gorm:"primaryKey;autoIncrement:false"` // Telegram Chat ID
	FirstName string `gorm:"size:255"`
	UserName  string `gorm:"size:255"`

	// Finite State Machine
	State string `gorm:"size:50;default:'idle'"`

	// Draft fields for Wizard (temporary storage during setup)
	DraftName   string `gorm:"size:255"`
	DraftBroker string `gorm:"size:255"`
	DraftTopic  string `gorm:"size:255"`
	DraftKey    string `gorm:"size:255"`

	Targets []MonitoringTarget `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type MonitoringTarget struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    int64  `gorm:"index"`
	Name      string `gorm:"size:255"` // Friendly name (e.g. "CNC 1")
	Broker    string `gorm:"size:255"`
	Topic     string `gorm:"size:255"`
	Key       string `gorm:"size:255"` // Optional Kafka Key (e.g. "192.168.1.10:8193")
	CreatedAt time.Time
}
