package models

import "encoding/json"

// FanucMessage представляет структуру сообщения, получаемого из топика Kafka.
// Она должна соответствовать тому, что отправляет fanucService.
type FanucMessage struct {
	MachineID string          `json:"machine_id"`
	Timestamp int64           `json:"timestamp"` // Unix ms, если отправляется
	Data      json.RawMessage `json:"data"`      // Сырые данные (Focas data), структуру которых мы можем уточнить позже
}
