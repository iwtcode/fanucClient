package telegram

import (
	"log"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func LogMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			user := c.Sender()
			username := user.FirstName
			if user.Username != "" {
				username += " (@" + user.Username + ")"
			}

			// Определяем контент сообщения
			content := strings.TrimSpace(c.Text())
			prefix := "TEXT"

			if cb := c.Callback(); cb != nil {
				prefix = "BTN"
				// Сначала пробуем взять Data (она содержит payload)
				content = strings.TrimSpace(cb.Data)

				// Если Data пустая (такое бывает у статических кнопок), берем Unique
				if content == "" {
					content = strings.TrimSpace(cb.Unique)
				}

				// Если в Data содержится Unique (дублирование), оставляем только Data,
				// но если они разные, можно залогировать как "Unique | Payload"
				// В данном случае просто берем непустую строку.
			}

			// Логируем в формате: [ID] Username | TYPE: Content (Duration)
			log.Printf("[%d] %s | %s: %s (%v)",
				user.ID,
				username,
				prefix,
				content,
				duration,
			)

			return err
		}
	}
}
