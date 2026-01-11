package main

import (
	"github.com/iwtcode/fanucClient/internal/app"
)

func main() {
	// Запуск приложения через DI контейнер (fx)
	app.New().Run()
}
