package services

import (
	"context"
	"fmt"
	"time"

	"github.com/iwtcode/fanucClient/internal/interfaces"
	"github.com/segmentio/kafka-go"
)

type kafkaService struct{}

func NewKafkaService() interfaces.KafkaReader {
	return &kafkaService{}
}

func (s *kafkaService) GetLastMessage(ctx context.Context, broker, topic string) (string, error) {
	if broker == "" || topic == "" {
		return "", fmt.Errorf("broker or topic is empty")
	}

	// Устанавливаем таймаут для соединения
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. Подключаемся к лидеру партиции 0 (упрощение: считаем, что данные в 0 партиции)
	conn, err := kafka.DialLeader(dialCtx, "tcp", broker, topic, 0)
	if err != nil {
		return "", fmt.Errorf("failed to dial leader: %w", err)
	}
	defer conn.Close()

	// 2. Получаем последний оффсет (конец очереди)
	lastOffset, err := conn.ReadLastOffset()
	if err != nil {
		return "", fmt.Errorf("failed to read last offset: %w", err)
	}

	if lastOffset == 0 {
		return "⚠️ Topic is empty", nil
	}

	// 3. Смещаемся на один шаг назад, чтобы прочитать последнее сообщение
	// Важно: SeekAbsolute. Seek возвращает (int64, error).
	if _, err := conn.Seek(lastOffset-1, kafka.SeekAbsolute); err != nil {
		return "", fmt.Errorf("failed to seek: %w", err)
	}

	// 4. Читаем сообщение
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	batch := conn.ReadBatch(1, 1) // min=1, max=1 bytes (но сообщение вычитается целиком)
	defer batch.Close()

	msg, err := batch.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to read message: %w", err)
	}

	return string(msg.Value), nil
}
