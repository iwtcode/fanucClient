package usecases

import (
	"errors"
	"strings"

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

func (u *settingsUsecase) SetContextSvcID(userID int64, svcID uint) error {
	return u.repo.UpdateDraft(userID, map[string]interface{}{"context_svc_id": svcID})
}
func (u *settingsUsecase) SetContextMachineID(userID int64, machineID string) error {
	return u.repo.UpdateDraft(userID, map[string]interface{}{"context_machine_id": machineID})
}
func (u *settingsUsecase) SetContextTargetID(userID int64, targetID uint) error {
	return u.repo.UpdateDraft(userID, map[string]interface{}{"context_target_id": targetID})
}

func (u *settingsUsecase) SetDraftConnEndpoint(userID int64, endpoint string) error {
	return u.repo.UpdateDraft(userID, map[string]interface{}{
		"draft_conn_endpoint": endpoint,
		"state":               entities.StateWaitingConnTimeout,
	})
}
func (u *settingsUsecase) SetDraftConnTimeout(userID int64, timeout int) error {
	return u.repo.UpdateDraft(userID, map[string]interface{}{
		"draft_conn_timeout": timeout,
		"state":              entities.StateWaitingConnModel,
	})
}
func (u *settingsUsecase) SetDraftConnModel(userID int64, model string) error {
	return u.repo.UpdateDraft(userID, map[string]interface{}{
		"draft_conn_model": model,
		"state":            entities.StateWaitingConnSeries,
	})
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
func (u *settingsUsecase) SetDraftTopicAndSave(id int64, topic string) error {
	user, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	target := &entities.MonitoringTarget{
		UserID: user.ID,
		Name:   user.DraftName,
		Broker: user.DraftBroker,
		Topic:  topic,
	}
	if err := u.repo.AddTarget(target); err != nil {
		return err
	}
	return u.repo.UpdateState(id, entities.StateIdle)
}

// --- Direct API for Web ---

func (u *settingsUsecase) CreateTargetDirect(userID int64, name, broker, topic string) error {
	target := &entities.MonitoringTarget{
		UserID: userID,
		Name:   name,
		Broker: broker,
		Topic:  topic,
	}
	return u.repo.AddTarget(target)
}

func (u *settingsUsecase) UpdateTargetDirect(userID int64, targetID uint, name, broker, topic string) error {
	target, err := u.repo.GetTargetByID(targetID)
	if err != nil || target.UserID != userID {
		return errors.New("target not found or access denied")
	}
	target.Name = name
	target.Broker = broker
	target.Topic = topic
	return u.repo.UpdateTarget(target)
}

func (u *settingsUsecase) CreateServiceDirect(userID int64, name, host, key string) error {
	host = strings.TrimSpace(host)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	svc := &entities.FanucService{
		UserID:  userID,
		Name:    name,
		BaseURL: host,
		APIKey:  key,
	}
	return u.repo.AddService(svc)
}

func (u *settingsUsecase) UpdateServiceDirect(userID int64, svcID uint, name, host, key string) error {
	svc, err := u.repo.GetServiceByID(svcID)
	if err != nil || svc.UserID != userID {
		return errors.New("service not found or access denied")
	}
	host = strings.TrimSpace(host)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	svc.Name = name
	svc.BaseURL = host
	svc.APIKey = key
	return u.repo.UpdateService(svc)
}

func (u *settingsUsecase) AddKeyToTargetDirect(targetID uint, key string) error {
	newKey := &entities.MonitoringKey{
		TargetID: targetID,
		Key:      key,
	}
	return u.repo.AddKey(newKey)
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

func (u *settingsUsecase) AddKeyToTarget(userID int64, key string) error {
	user, err := u.repo.GetByID(userID)
	if err != nil {
		return err
	}
	newKey := &entities.MonitoringKey{
		TargetID: user.ContextTargetID,
		Key:      key,
	}
	if err := u.repo.AddKey(newKey); err != nil {
		return err
	}
	return u.repo.UpdateState(userID, entities.StateIdle)
}
func (u *settingsUsecase) DeleteKey(keyID uint) error {
	return u.repo.DeleteKey(keyID)
}
func (u *settingsUsecase) GetKeyByID(keyID uint) (*entities.MonitoringKey, error) {
	return u.repo.GetKeyByID(keyID)
}

func (u *settingsUsecase) SetDraftSvcName(id int64, name string) error {
	return u.repo.UpdateDraft(id, map[string]interface{}{
		"draft_svc_name": name,
		"state":          entities.StateWaitingSvcHost,
	})
}
func (u *settingsUsecase) SetDraftSvcHost(id int64, host string) error {
	return u.repo.UpdateDraft(id, map[string]interface{}{
		"draft_svc_host": host,
		"state":          entities.StateWaitingSvcKey,
	})
}
func (u *settingsUsecase) SetDraftSvcKeyAndSave(id int64, key string) error {
	user, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	host := strings.TrimSpace(user.DraftSvcHost)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "http://" + host
	}
	svc := &entities.FanucService{
		UserID:  user.ID,
		Name:    user.DraftSvcName,
		BaseURL: host,
		APIKey:  key,
	}
	if err := u.repo.AddService(svc); err != nil {
		return err
	}
	return u.repo.UpdateState(id, entities.StateIdle)
}

func (u *settingsUsecase) GetServices(userID int64) ([]entities.FanucService, error) {
	return u.repo.GetServices(userID)
}
func (u *settingsUsecase) DeleteService(userID int64, svcID uint) error {
	return u.repo.DeleteService(svcID, userID)
}
func (u *settingsUsecase) GetServiceByID(svcID uint) (*entities.FanucService, error) {
	return u.repo.GetServiceByID(svcID)
}
