package fanucClient

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

var (
	// Время запуска бота для Uptime
	botStartTime = time.Now()

	// Нижнее меню
	menu    = &tele.ReplyMarkup{ResizeKeyboard: true}
	btnPing = menu.Text("🏓 Ping")
	btnWho  = menu.Text("👤 Info")
	btnTime = menu.Text("⏰ Time")

	// Инлайн меню
	inlineMenu    = &tele.ReplyMarkup{}
	btnPingInline = inlineMenu.Data("🏓 Ping", "ping_btn")
	btnWhoInline  = inlineMenu.Data("👤 Info", "who_btn")
	btnTimeInline = inlineMenu.Data("⏰ Time", "time_btn")
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	pref := tele.Settings{
		Token:     os.Getenv("TG_TOKEN"),
		Poller:    &tele.LongPoller{Timeout: 10 * time.Second},
		ParseMode: tele.ModeHTML,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	b.Use(middleware.Recover())

	b.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user := c.Sender()
			text := c.Text()
			if cb := c.Callback(); cb != nil {
				unique := strings.TrimSpace(cb.Unique)
				label := unique
				switch unique {
				case btnPingInline.Unique:
					label = btnPingInline.Text
				case btnWhoInline.Unique:
					label = btnWhoInline.Text
				case btnTimeInline.Unique:
					label = btnTimeInline.Text
				}
				text = "[Inline] " + label
			} else {
				if text == btnPing.Text || text == btnWho.Text || text == btnTime.Text {
					text = "[menu]" + text
				}
			}
			log.Printf("[%d] %s: %s", user.ID, user.FirstName, text)
			return next(c)
		}
	})

	menu.Reply(
		menu.Row(btnPing, btnWho),
		menu.Row(btnTime),
	)

	inlineMenu.Inline(
		inlineMenu.Row(btnPingInline, btnWhoInline, btnTimeInline),
	)

	pingFunc := func(c tele.Context) error {
		start := time.Now()
		resp, err := http.Head("https://api.telegram.org")
		if err != nil {
			return refreshMessage(c, fmt.Sprintf("🏓 <b>Pong!</b>\n\n❌ Error: %s", err.Error()))
		}
		defer resp.Body.Close()

		latency := time.Since(start).Milliseconds()
		uptime := time.Since(botStartTime).Round(time.Second)
		statusIcon := "🟢"
		if latency > 200 {
			statusIcon = "🟠"
		}

		text := fmt.Sprintf(
			"🏓 <b>Pong!</b>\n\n"+
				"%s Network: <code>%d ms</code>\n"+
				"⏳ Uptime: <code>%s</code>\n"+
				"📅 Checked: %s",
			statusIcon,
			latency,
			uptime.String(),
			time.Now().Format("15:04:05"),
		)

		return refreshMessage(c, text)
	}

	whoFunc := func(c tele.Context) error {
		u := c.Sender()
		text := fmt.Sprintf("👤 <b>User Information</b>\n\n🆔 ID: <code>%d</code>\n📛 Name: <b>%s</b>\n🌐 Lang: %s",
			u.ID, u.FirstName, u.LanguageCode)
		return refreshMessage(c, text)
	}

	timeFunc := func(c tele.Context) error {
		now := time.Now()
		text := fmt.Sprintf("⏰ <b>Server Time</b>\n\n📅 Date: <b>%s</b>\n⌚ Time: <b>%s</b>\n🌍 Zone: %s",
			now.Format("02 Jan 2006"),
			now.Format("15:04:05"),
			now.Location().String(),
		)
		return refreshMessage(c, text)
	}

	b.Handle("/start", func(c tele.Context) error {
		text := fmt.Sprintf("👋 <b>Панель управления</b>\n\nПривет, %s!", c.Sender().FirstName)
		return refreshMessage(c, text)
	})

	b.Handle("/ping", pingFunc)
	b.Handle("/whoami", whoFunc)
	b.Handle("/time", timeFunc)

	b.Handle(&btnPing, pingFunc)
	b.Handle(&btnWho, whoFunc)
	b.Handle(&btnTime, timeFunc)

	b.Handle(&btnPingInline, pingFunc)
	b.Handle(&btnWhoInline, whoFunc)
	b.Handle(&btnTimeInline, timeFunc)

	log.Println("🔥 Бот запущен")
	b.Start()
}

func refreshMessage(c tele.Context, text string) error {
	if c.Callback() != nil {
		c.Respond()
		return c.Edit(text, inlineMenu)
	}
	return c.Send(text, inlineMenu)
}
