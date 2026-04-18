package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/domain/models"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type ConsumerManager struct {
	db       *gorm.DB
	notifier interfaces.NotifierService

	readers map[string]*kafka.Reader // ключ: "broker|topic"
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex

	// Дебаунс: userID -> machineID -> hash состояния (чтобы не спамить одно и то же)
	alarmStates map[int64]map[string]string
	stateMu     sync.RWMutex
}

func NewConsumerManager(db *gorm.DB, notifier interfaces.NotifierService) *ConsumerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ConsumerManager{
		db:          db,
		notifier:    notifier,
		readers:     make(map[string]*kafka.Reader),
		ctx:         ctx,
		cancel:      cancel,
		alarmStates: make(map[int64]map[string]string),
	}
}

func (m *ConsumerManager) Start() {
	log.Println("🛠 Запуск фонового воркера Kafka Consumer...")
	go m.syncLoop()
}

func (m *ConsumerManager) Stop() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.readers {
		r.Close()
	}
	log.Println("🛑 Kafka Consumer воркер остановлен.")
}

// syncLoop раз в 30 секунд обновляет список таргетов из БД
func (m *ConsumerManager) syncLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	m.updateReaders() // первый запуск сразу

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateReaders()
		}
	}
}

func (m *ConsumerManager) updateReaders() {
	var targets []entities.MonitoringTarget
	if err := m.db.Preload("Keys").Find(&targets).Error; err != nil {
		log.Printf("❌ Ошибка получения Targets для воркера: %v", err)
		return
	}

	activeKeys := make(map[string]bool)

	for _, t := range targets {
		key := fmt.Sprintf("%s|%s", t.Broker, t.Topic)
		activeKeys[key] = true

		m.mu.Lock()
		if _, exists := m.readers[key]; !exists {
			log.Printf("✅ Запуск прослушивания топика: %s (Broker: %s)", t.Topic, t.Broker)
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:     []string{t.Broker},
				Topic:       t.Topic,
				StartOffset: kafka.LastOffset, // Читаем только новые сообщения
				MaxBytes:    10e6,             // 10MB
			})
			m.readers[key] = r
			go m.consumeRoutine(r, t.Broker, t.Topic)
		}
		m.mu.Unlock()
	}

	// Очистка старых ридеров (если таргет удалили)
	m.mu.Lock()
	for key, r := range m.readers {
		if !activeKeys[key] {
			log.Printf("🗑 Остановка прослушивания: %s", key)
			r.Close()
			delete(m.readers, key)
		}
	}
	m.mu.Unlock()
}

func (m *ConsumerManager) consumeRoutine(r *kafka.Reader, broker, topic string) {
	for {
		msg, err := r.ReadMessage(m.ctx)
		if err != nil {
			if m.ctx.Err() != nil {
				return // контекст отменен
			}
			log.Printf("⚠️ Ошибка чтения из Kafka (%s/%s): %v", broker, topic, err)
			time.Sleep(5 * time.Second)
			continue
		}

		m.processMessage(broker, topic, msg.Key, msg.Value)
	}
}

func (m *ConsumerManager) processMessage(broker, topic string, msgKey, msgValue []byte) {
	var fanucMsg models.FanucMessage
	if err := json.Unmarshal(msgValue, &fanucMsg); err != nil {
		return // Пропускаем невалидный JSON
	}

	// Ищем пользователей, которые подписаны на этот брокер/топик
	var targets []entities.MonitoringTarget
	m.db.Preload("Keys").Where("broker = ? AND topic = ?", broker, topic).Find(&targets)

	for _, t := range targets {
		userID := t.UserID

		// Проверка ключа (если у пользователя заданы ключи, сообщение должно совпадать хотя бы с одним)
		if len(t.Keys) > 0 {
			matched := false
			strKey := string(msgKey)
			for _, k := range t.Keys {
				if k.Key == strKey {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		m.evaluateAlert(userID, fanucMsg)
	}
}

func (m *ConsumerManager) evaluateAlert(userID int64, msg models.FanucMessage) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	if m.alarmStates[userID] == nil {
		m.alarmStates[userID] = make(map[string]string)
	}

	// Генерируем хэш текущего состояния ошибок (чтобы определить, изменилось ли оно)
	stateBytes, _ := json.Marshal(msg.Alarms)
	currentHash := ""
	if msg.IsEmergency || msg.HasAlarms {
		hash := sha256.Sum256(stateBytes)
		currentHash = hex.EncodeToString(hash[:])
		if msg.IsEmergency {
			currentHash += "_E" // Добавляем метку emergency
		}
	}

	lastHash := m.alarmStates[userID][msg.MachineID]

	// Состояние не изменилось -> ничего не делаем
	if lastHash == currentHash {
		return
	}

	// Обновляем хэш
	m.alarmStates[userID][msg.MachineID] = currentHash

	// Формируем payload
	payload := models.AlertPayload{
		MachineID: msg.MachineID,
		Alarms:    msg.Alarms,
	}

	if currentHash == "" && lastHash != "" {
		// Была ошибка, теперь ее нет -> Resolved
		payload.Type = "resolved"
		payload.Message = "Все ошибки и состояния аварийной остановки устранены."
		m.notifier.NotifyAlarm(userID, payload)
	} else if msg.IsEmergency {
		payload.Type = "emergency"
		payload.Message = "Сработала аварийная остановка (Emergency Stop)!"
		m.notifier.NotifyAlarm(userID, payload)
	} else if msg.HasAlarms {
		payload.Type = "alarm"
		payload.Message = fmt.Sprintf("Обнаружено %d активных ошибок.", len(msg.Alarms))
		m.notifier.NotifyAlarm(userID, payload)
	}
}
