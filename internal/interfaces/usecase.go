package interfaces

import (
	"context"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucService"
)

type SettingsUsecase interface {
	RegisterUser(user *entities.User) error
	GetUser(id int64) (*entities.User, error)
	SetState(id int64, state string) error

	// Context Helpers for Wizards
	SetContextSvcID(userID int64, svcID uint) error
	SetContextMachineID(userID int64, machineID string) error
	SetContextTargetID(userID int64, targetID uint) error

	// Connection Wizard Steps
	SetDraftConnEndpoint(userID int64, endpoint string) error
	SetDraftConnTimeout(userID int64, timeout int) error
	SetDraftConnModel(userID int64, model string) error

	// Kafka Targets Management
	SetDraftName(id int64, name string) error
	SetDraftBroker(id int64, broker string) error
	SetDraftTopicAndSave(id int64, topic string) error

	GetTargets(userID int64) ([]entities.MonitoringTarget, error)
	DeleteTarget(userID int64, targetID uint) error
	GetTargetByID(targetID uint) (*entities.MonitoringTarget, error)

	// Kafka Key Management
	AddKeyToTarget(userID int64, key string) error
	DeleteKey(keyID uint) error
	GetKeyByID(keyID uint) (*entities.MonitoringKey, error)

	// Fanuc Services Management
	SetDraftSvcName(id int64, name string) error
	SetDraftSvcHost(id int64, host string) error
	SetDraftSvcKeyAndSave(id int64, key string) error
	GetServices(userID int64) ([]entities.FanucService, error)
	DeleteService(userID int64, svcID uint) error
	GetServiceByID(svcID uint) (*entities.FanucService, error)
}

type MonitoringUsecase interface {
	// keyID == 0 means "no key" (default)
	// Returns: foundKey, foundValue, error
	FetchLastKafkaMessage(ctx context.Context, targetID uint, keyID uint) (string, string, error)
}

type ControlUsecase interface {
	// Machine Management
	CreateMachine(ctx context.Context, svcID uint, req fanucService.ConnectionRequest) (*fanucService.MachineDTO, error)
	ListMachines(ctx context.Context, svcID uint) ([]fanucService.MachineDTO, error)
	GetMachine(ctx context.Context, svcID uint, machineID string) (*fanucService.MachineDTO, error)
	DeleteMachine(ctx context.Context, svcID uint, machineID string) error

	// Actions
	StartPolling(ctx context.Context, svcID uint, machineID string, intervalMs int) error
	StopPolling(ctx context.Context, svcID uint, machineID string) error
	GetProgram(ctx context.Context, svcID uint, machineID string) (string, error)
}
