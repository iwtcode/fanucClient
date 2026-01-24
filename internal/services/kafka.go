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
		return "", "", fmt.Errorf("broker or topic is empty")
	}

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. Connect to partition 0 leader (Assuming single partition for simplicity or 0)
	conn, err := kafka.DialLeader(dialCtx, "tcp", broker, topic, 0)
	if err != nil {
		return "", "", fmt.Errorf("failed to dial leader: %w", err)
	}
	defer conn.Close()

	// 2. Get last offset
	lastOffset, err := conn.ReadLastOffset()
	if err != nil {
		return "", "", fmt.Errorf("failed to read last offset: %w", err)
	}

	if lastOffset == 0 {
		return "", "⚠️ Topic is empty", nil
	}

	// 3. Determine scan range
	// If key is present, we scan last 1000 messages to find it.
	// If key is empty, we just take the last message.
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

	// 4. Read batch
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	batch := conn.ReadBatch(10e3, 1e6) // min 10KB, max 1MB
	defer batch.Close()

	var foundMsg *kafka.Message

	// Scan through messages from startOffset to lastOffset
	for {
		m, err := batch.ReadMessage()
		if err != nil {
			break // Batch finished or error
		}

		if keyFilter != "" {
			// If looking for a key, update foundMsg only if key matches
			if string(m.Key) == keyFilter {
				// We create a copy because m is reused in loop
				msgCopy := m
				foundMsg = &msgCopy
			}
		} else {
			// If no key, just take the last one seen
			msgCopy := m
			foundMsg = &msgCopy
		}

		// If we reached the end
		if m.Offset >= lastOffset-1 {
			break
		}
	}

	if foundMsg == nil {
		if keyFilter != "" {
			return "", fmt.Sprintf("⚠️ Message with key '%s' not found in last %d messages", keyFilter, scanDepth), nil
		}
		return "", "⚠️ Could not read message", nil
	}

	return string(foundMsg.Key), string(foundMsg.Value), nil
}
