package models

import "encoding/json"

// AlarmDetail соответствует структуре отправляемой из fanucService
type AlarmDetail struct {
	ErrorCode            string `json:"error_code"`
	ErrorTypeDescription string `json:"error_type_description"`
	ErrorMessage         string `json:"error_message"`
}

// FanucMessage представляет структуру сообщения, получаемого из топика Kafka.
type FanucMessage struct {
	MachineID   string          `json:"machine_id"`
	Timestamp   string          `json:"timestamp"`
	IsEmergency bool            `json:"is_emergency"`
	HasAlarms   bool            `json:"has_alarms"`
	Alarms      []AlarmDetail   `json:"alarms"`
	RawData     json.RawMessage `json:"-"` // Для внутреннего использования
}

// AlertPayload структура для отправки на фронтенд (Web)
type AlertPayload struct {
	MachineID string        `json:"machine_id"`
	Type      string        `json:"type"` // "alarm", "emergency", "resolved"
	Message   string        `json:"message"`
	Alarms    []AlarmDetail `json:"alarms"`
}
