# fanucClient

```
fanucClient/
├── cmd/bot/
│   └── main.go                             # Точка входа. Инициализирует конфигурацию и запускает fx.App (DI контейнер)
│
├── internal/
│   ├── app/
│   │   └── app.go                          # Сборка зависимостей (Dependency Injection).   
│   │
│   ├── domain/                             # Cлой данных (Data Layer)
│   │   ├── entities/
│   │   │   └── user.go                     # GORM модель пользователя (ID, Kafka Endpoint, API Key, Fanuc URL)
│   │   └── models/
│   │       ├── commands.go                 # Внутренние модели для передачи команд управления (DTO)
│   │       └── events.go                   # Модели событий, получаемых из Kafka (DTO)
│   │
│   ├── handlers/                           # Транспортный слой (Delivery Layer)
│   │   ├── telegram/                       # Обработка взаимодействий с Telegram (аналог HTTP контроллеров)
│   │   │   ├── bot.go                      # Инициализация Telebot
│   │   │   ├── middleware.go               # Логирование, Auth, Recover
│   │   │   ├── router.go                   # Регистрация хендлеров и кнопок
│   │   │   ├── menu.go                     # Определение клавиатур и меню
│   │   │   ├── commands.go                 # Обработчики команд (/start, /settings)
│   │   │   └── callbacks.go                # Обработчики нажатий на кнопки
│   │   └── worker/                         # Фоновые процессы
│   │       └── consumer.go                 # Обработчик, который слушает Kafka Consumer и передает данные в Usecase
│   │
│   ├── interfaces/                         # Контракты (Абстракции)
│   │   ├── repository.go                   # Интерфейс для работы с БД (сохранение/чтение пользователей)
│   │   ├── service.go                      # Интерфейсы внешних сервисов (FanucAPI, KafkaReader, NotificationSender)
│   │   └── usecase.go                      # Интерфейсы бизнес-логики (Monitoring, Control, Settings)
│   │
│   ├── repository/                         # Реализация доступа к данным (Adapter)
│   │   ├── postgres.go                     # Подключение к PostgreSQL, настройка GORM и миграции
│   │   └── user.go                         # Реализация методов интерфейса Repository для сущности User
│   │
│   ├── services/                           # Реализация внешних сервисов (Infrastructure)
│   │   ├── fanuc.go                        # Обертка над client.go, реализующая интерфейс для управления станком через HTTP
│   │   ├── kafka.go                        # Реализация Kafka Consumer (чтение сообщений из топиков)
│   │   └── notifier.go                     # Сервис отправки уведомлений
│   │
│   └── usecases/                           # Слой бизнес-логики (Application Business Rules)
│       ├── control.go                      # Логика управления: вызов команд fanuc-сервиса
│       ├── monitoring.go                   # Логика мониторинга: анализ данных из Kafka, принятие решения об отправке алерта
│       └── settings.go                     # Логика настроек: сохранение/обновление API ключей и эндпоинтов пользователя
│
├── .env.example                            # Шаблон переменных окружения
├── client.go                               # SDK клиент для fanucService
├── config.go                               # Загрузка конфигурации из .env в структуру Config
├── docker-compose.yml                      # Инфраструктура для локального запуска (PostgreSQL, Kafka для тестов)
└── models.go                               # Общие модели данных (Shared Models), используемые и в SDK, и внутри
```