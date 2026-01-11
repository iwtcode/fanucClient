package interfaces

import "context"

type KafkaReader interface {
	// GetLastMessage подключается к указанному брокеру и топику,
	// и возвращает последнее записанное сообщение.
	GetLastMessage(ctx context.Context, broker, topic string) (string, error)
}
