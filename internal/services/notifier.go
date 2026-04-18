package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"

	"github.com/iwtcode/fanucClient/internal/domain/models"
	"github.com/iwtcode/fanucClient/internal/interfaces"
)

type notifierService struct {
	tgBot  interfaces.TelegramSender
	webSrv interfaces.WebSender
}

func NewNotifierService(tgBot interfaces.TelegramSender, webSrv interfaces.WebSender) interfaces.NotifierService {
	return &notifierService{
		tgBot:  tgBot,
		webSrv: webSrv,
	}
}

func (s *notifierService) NotifyAlarm(userID int64, payload models.AlertPayload) {
	// 1. Отправка в Web через SSE
	jsonData, _ := json.Marshal(payload)
	s.webSrv.BroadcastToUser(userID, "alert", jsonData)

	// 2. Отправка в Telegram
	var tgText bytes.Buffer

	machineSafe := html.EscapeString(payload.MachineID)
	msgSafe := html.EscapeString(payload.Message)

	switch payload.Type {
	case "emergency":
		tgText.WriteString("🚨 <b>EMERGENCY STOP</b> 🚨\n\n")
		tgText.WriteString(fmt.Sprintf("Станок: <code>%s</code>\n", machineSafe))
		tgText.WriteString(fmt.Sprintf("Сообщение: %s\n", msgSafe))
	case "alarm":
		tgText.WriteString("⚠️ <b>ALARM</b> ⚠️\n\n")
		tgText.WriteString(fmt.Sprintf("Станок: <code>%s</code>\n", machineSafe))
		tgText.WriteString(fmt.Sprintf("Сообщение: %s\n", msgSafe))
	case "resolved":
		tgText.WriteString("✅ <b>ПРОБЛЕМА УСТРАНЕНА</b>\n\n")
		tgText.WriteString(fmt.Sprintf("Станок: <code>%s</code>\n", machineSafe))
		tgText.WriteString(fmt.Sprintf("Сообщение: %s\n", msgSafe))
	}

	if len(payload.Alarms) > 0 {
		tgText.WriteString("\n<b>Детали ошибок:</b>\n")
		for _, a := range payload.Alarms {
			tgText.WriteString(fmt.Sprintf("• [%s] <b>%s</b>: %s\n",
				html.EscapeString(a.ErrorCode),
				html.EscapeString(a.ErrorTypeDescription),
				html.EscapeString(a.ErrorMessage)))
		}
	}

	// Telegram отправка
	_ = s.tgBot.SendAlert(userID, tgText.String())
}
