package entities

import (
	"time"
)

// State constants for FSM
const (
	StateIdle = "idle"

	// Kafka Target Wizard
	StateWaitingName   = "waiting_name"
	StateWaitingBroker = "waiting_broker"
	StateWaitingTopic  = "waiting_topic"
	StateWaitingKey    = "waiting_key"

	// Service Wizard
	StateWaitingSvcName = "waiting_svc_name"
	StateWaitingSvcHost = "waiting_svc_host" // IP:PORT
	StateWaitingSvcKey  = "waiting_svc_key"  // API Key
)

type User struct {
	ID        int64  `gorm:"primaryKey;autoIncrement:false"` // Telegram Chat ID
	FirstName string `gorm:"size:255"`
	UserName  string `gorm:"size:255"`

	// Finite State Machine
	State string `gorm:"size:50;default:'idle'"`

	// Draft fields for Kafka Wizard
	DraftName   string `gorm:"size:255"`
	DraftBroker string `gorm:"size:255"`
	DraftTopic  string `gorm:"size:255"`
	DraftKey    string `gorm:"size:255"`

	// Draft fields for Service Wizard
	DraftSvcName string `gorm:"size:255"`
	DraftSvcHost string `gorm:"size:255"`
	DraftSvcKey  string `gorm:"size:255"`

	// Relations
	Targets  []MonitoringTarget `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Services []FanucService     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// MonitoringTarget - подключение к Kafka (чтение)
type MonitoringTarget struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    int64  `gorm:"index"`
	Name      string `gorm:"size:255"`
	Broker    string `gorm:"size:255"`
	Topic     string `gorm:"size:255"`
	Key       string `gorm:"size:255"`
	CreatedAt time.Time
}

// FanucService - подключение к REST API fanucService (управление)
type FanucService struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    int64  `gorm:"index"`
	Name      string `gorm:"size:255"` // Friendly name (e.g. "Цех №1")
	BaseURL   string `gorm:"size:255"` // http://ip:port
	APIKey    string `gorm:"size:255"`
	CreatedAt time.Time
}
