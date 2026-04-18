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

func (s *kafkaService) GetLastMessage(ctx context.Context, broker, topic, keyFilter string) (string, string, error) {
	if broker == "" || topic == "" {
		return "", "", fmt.Errorf("broker или topic пусты")
	}

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. Подключаемся к лидеру партиции 0
	conn, err := kafka.DialLeader(dialCtx, "tcp", broker, topic, 0)
	if err != nil {
		return "", "", fmt.Errorf("failed to dial leader: %w", err)
	}
	defer conn.Close()

	// 2. Получаем High Watermark (следующий оффсет, который будет записан)
	lastOffset, err := conn.ReadLastOffset()
	if err != nil {
		return "", "", fmt.Errorf("failed to read last offset: %w", err)
	}

	if lastOffset == 0 {
		return "", "⚠️ Топик пуст", nil
	}

	// 3. Определяем глубину поиска
	scanDepth := int64(1)
	if keyFilter != "" {
		scanDepth = 1000
	}

	startOffset := lastOffset - scanDepth
	if startOffset < 0 {
		startOffset = 0
	}

	if _, err := conn.Seek(startOffset, kafka.SeekAbsolute); err != nil {
		return "", "", fmt.Errorf("failed to seek: %w", err)
	}

	// 4. Читаем данные
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	batch := conn.ReadBatch(1, 1e6) // min 1 Byte, max 1MB
	defer batch.Close()

	var foundMsg *kafka.Message

	// Сканируем сообщения от startOffset до lastOffset
	for {
		m, err := batch.ReadMessage()
		if err != nil {
			break // Конец батча или ошибка/таймаут
		}

		if keyFilter != "" {
			// Если ищем по ключу, обновляем foundMsg только при совпадении
			if string(m.Key) == keyFilter {
				msgCopy := m
				foundMsg = &msgCopy
			}
		} else {
			// Если без ключа, просто берем последнее увиденное
			msgCopy := m
			foundMsg = &msgCopy
		}

		// Прерываем цикл, как только прочитали самое последнее существующее сообщение (lastOffset - 1)
		if m.Offset >= lastOffset-1 {
			break
		}
	}

	if foundMsg == nil {
		if keyFilter != "" {
			return "", fmt.Sprintf("⚠️ Сообщение с ключом '%s' не найдено в последних %d записях", keyFilter, scanDepth), nil
		}
		return "", "⚠️ Не удалось прочитать сообщение", nil
	}

	return string(foundMsg.Key), string(foundMsg.Value), nil
}
