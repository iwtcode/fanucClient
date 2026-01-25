package telegram

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"github.com/iwtcode/fanucClient/internal/interfaces"
	"github.com/iwtcode/fanucService"
	tele "gopkg.in/telebot.v3"
)

type CommandHandler struct {
	menu       *Menu
	settingsUC interfaces.SettingsUsecase
	controlUC  interfaces.ControlUsecase
}

func NewCommandHandler(
	menu *Menu,
	settingsUC interfaces.SettingsUsecase,
	controlUC interfaces.ControlUsecase,
) *CommandHandler {
	return &CommandHandler{
		menu:       menu,
		settingsUC: settingsUC,
		controlUC:  controlUC,
	}
}

func (h *CommandHandler) OnStart(c tele.Context) error {
	h.settingsUC.SetState(c.Sender().ID, entities.StateIdle)

	user := &entities.User{
		ID:        c.Sender().ID,
		FirstName: c.Sender().FirstName,
		UserName:  c.Sender().Username,
		State:     entities.StateIdle,
	}
	h.settingsUC.RegisterUser(user)

	text := "👋 <b>Fanuc Client</b>\n\nГлавное меню."
	if c.Callback() != nil {
		return c.Edit(text, h.menu.BuildMainMenu())
	}
	return c.Send(text, h.menu.ReplyMain, h.menu.BuildMainMenu())
}

func (h *CommandHandler) OnWho(c tele.Context) error {
	u, err := h.settingsUC.GetUser(c.Sender().ID)
	if err != nil {
		return c.Send("Ошибка получения пользователя")
	}
	text := fmt.Sprintf("👤 <b>Профиль</b>\nID: <code>%d</code>\nСостояние: <code>%s</code>", u.ID, u.State)

	targets, _ := h.settingsUC.GetTargets(u.ID)
	services, _ := h.settingsUC.GetServices(u.ID)

	text += fmt.Sprintf("\n\n📋 Kafka Targets: %d", len(targets))
	text += fmt.Sprintf("\n🌐 API Services: %d", len(services))

	if c.Callback() != nil {
		return c.Edit(text, h.menu.BuildWhoMenu())
	}
	return c.Send(text, h.menu.BuildWhoMenu())
}

// OnKafka обрабатывает команду /kafka из меню
func (h *CommandHandler) OnKafka(c tele.Context) error {
	userID := c.Sender().ID
	h.settingsUC.SetState(userID, entities.StateIdle)

	targets, err := h.settingsUC.GetTargets(userID)
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Send("Ошибка получения Targets: " + safeErr)
	}

	text := fmt.Sprintf("📋 <b>Kafka Targets (%d)</b>\n\nВыберите <code>Kafka Target</code> для управления:", len(targets))
	markup := h.menu.BuildTargetsList(targets)

	return c.Send(text, markup)
}

// OnServices обрабатывает команду /services из меню
func (h *CommandHandler) OnServices(c tele.Context) error {
	userID := c.Sender().ID
	h.settingsUC.SetState(userID, entities.StateIdle)

	services, err := h.settingsUC.GetServices(userID)
	if err != nil {
		safeErr := html.EscapeString(err.Error())
		return c.Send("Ошибка получения сервисов: " + safeErr)
	}

	text := fmt.Sprintf("🌐 <b>Ваши сервисы (%d)</b>\n\nВыберите <code>API Service</code> для управления:", len(services))
	markup := h.menu.BuildServicesList(services)

	return c.Send(text, markup)
}

func (h *CommandHandler) OnText(c tele.Context) error {
	userID := c.Sender().ID
	user, err := h.settingsUC.GetUser(userID)
	if err != nil {
		return h.OnStart(c)
	}

	input := strings.TrimSpace(c.Text())

	// Menu Commands (Reply Keyboard)
	switch input {
	case h.menu.BtnHome.Text:
		return h.OnStart(c)
	case h.menu.BtnWho.Text:
		return h.OnWho(c)
	case h.menu.BtnTargets.Text:
		return h.OnKafka(c)
	case h.menu.BtnServices.Text:
		return h.OnServices(c)
	}

	// FSM Processing
	switch user.State {
	// --- Kafka Wizard ---
	case entities.StateWaitingName:
		h.settingsUC.SetDraftName(userID, input)
		return c.Send("🔌 <b>Шаг 2/3: Broker (IP:PORT)</b>", h.menu.BuildCancel())
	case entities.StateWaitingBroker:
		h.settingsUC.SetDraftBroker(userID, input)
		return c.Send("📂 <b>Шаг 3/3: Topic</b>", h.menu.BuildCancel())
	case entities.StateWaitingTopic:
		h.settingsUC.SetDraftTopicAndSave(userID, input)
		c.Send("✅ Kafka Target сохранен!")
		return h.OnKafka(c)

	// --- Adding Key to existing Target ---
	case entities.StateWaitingNewKey:
		h.settingsUC.AddKeyToTarget(userID, input)
		c.Send("✅ Ключ добавлен!")

		// Для редиректа на просмотр таргета нам нужен CallbackHandler.
		// Так как здесь мы в CommandHandler, мы просто вернем список таргетов.
		return h.OnKafka(c)

	// --- Service Registration Wizard ---
	case entities.StateWaitingSvcName:
		h.settingsUC.SetDraftSvcName(userID, input)
		return c.Send("🔗 <b>Шаг 2/3: Host (IP:PORT)</b>\nВведите адрес сервиса (без http://):", h.menu.BuildCancel())
	case entities.StateWaitingSvcHost:
		h.settingsUC.SetDraftSvcHost(userID, input)
		return c.Send("🔐 <b>Шаг 3/3: API Key</b>\nВведите ключ доступа к сервису:", h.menu.BuildCancel())
	case entities.StateWaitingSvcKey:
		h.settingsUC.SetDraftSvcKeyAndSave(userID, input)
		c.Send("✅ Сервис сохранен!")
		return h.OnServices(c)

	// --- Machine Connection Wizard (Remote API) ---
	case entities.StateWaitingConnEndpoint:
		h.settingsUC.SetDraftConnEndpoint(userID, input)
		return c.Send("⏱ <b>Шаг 2/4: Таймаут (мс)</b>\nВведите таймаут соединения (например 5000).\nОтправьте '0' или '-' для значения по умолчанию (5000ms).", h.menu.BuildCancel())

	case entities.StateWaitingConnTimeout:
		timeout := 5000
		if input != "0" && input != "-" {
			val, err := strconv.Atoi(input)
			if err != nil || val < 0 {
				return c.Send("⚠️ Введите корректное число или '-' для пропуска.")
			}
			timeout = val
		}
		h.settingsUC.SetDraftConnTimeout(userID, timeout)
		return c.Send("🤖 <b>Шаг 3/4: Модель</b>\nВведите название модели.\nОтправьте '0' или '-' для значения 'Unknown'.", h.menu.BuildCancel())

	case entities.StateWaitingConnModel:
		model := input
		if input == "0" || input == "-" {
			model = "Unknown"
		}
		h.settingsUC.SetDraftConnModel(userID, model)
		return c.Send("🔢 <b>Шаг 4/4: Серия</b>\nВведите серию стойки (0i, 30i, 31i).\nОтправьте '0' или '-' для значения 'Unknown'.", h.menu.BuildCancel())

	case entities.StateWaitingConnSeries:
		series := input
		if input == "0" || input == "-" {
			series = "Unknown"
		}

		svcID := user.ContextSvcID
		req := fanucService.ConnectionRequest{
			Endpoint: user.DraftConnEndpoint,
			Timeout:  user.DraftConnTimeout,
			Model:    user.DraftConnModel,
			Series:   series,
		}

		c.Send("⏳ Создание подключения на удаленном сервисе...")

		_, err := h.controlUC.CreateMachine(context.Background(), svcID, req)
		if err != nil {
			c.Send(fmt.Sprintf("❌ Ошибка создания подключения: %v", err))
		} else {
			c.Send("✅ Подключение установлено!")
		}

		h.settingsUC.SetState(userID, entities.StateIdle)
		// Возвращаемся в список сервисов
		return h.OnServices(c)

	// --- Polling Wizard ---
	case entities.StateWaitingPollInterval:
		interval, err := strconv.Atoi(input)
		if err != nil || interval < 100 {
			return c.Send("⚠️ Пожалуйста, введите корректное число (минимум 100 мс).")
		}

		svcID := user.ContextSvcID
		machineID := user.ContextMachineID

		c.Send("⏳ Запуск опроса...")
		err = h.controlUC.StartPolling(context.Background(), svcID, machineID, interval)
		if err != nil {
			c.Send(fmt.Sprintf("❌ Ошибка запуска опроса: %v", err))
		} else {
			c.Send("✅ Опрос запущен!")
		}

		h.settingsUC.SetState(userID, entities.StateIdle)
		return h.OnServices(c)

	default:
		return h.OnStart(c)
	}
}
