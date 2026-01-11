package interfaces

import "context"

type KafkaReader interface {
	// GetLastMessage подключается к указанному брокеру и топику.
	// Если key != "", ищет последнее сообщение с этим ключом (сканируя конец топика).
	// Если key == "", возвращает самое последнее сообщение в топике.
	GetLastMessage(ctx context.Context, broker, topic, key string) (string, error)
}
