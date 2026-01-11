package repository

import (
	"fmt"
	"log"

	"github.com/iwtcode/fanucClient"
	"github.com/iwtcode/fanucClient/internal/domain/entities"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresRepository(cfg *fanucClient.Config) *gorm.DB {
	// 1. Подключаемся к стандартной БД 'postgres', чтобы проверить/создать целевую БД
	dsnRoot := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword)

	rootDB, err := gorm.Open(postgres.Open(dsnRoot), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("failed to connect to root postgres db: %v", err)
	}

	var exists bool
	// Проверяем, существует ли база данных
	checkQuery := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", cfg.DBName)
	if err := rootDB.Raw(checkQuery).Scan(&exists).Error; err != nil {
		log.Fatalf("failed to check db existence: %v", err)
	}

	if !exists {
		log.Printf("Database %s does not exist. Creating...", cfg.DBName)
		// CREATE DATABASE не может работать внутри транзакции
		if err := rootDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)).Error; err != nil {
			log.Fatalf("failed to create database: %v", err)
		}
	}

	// Закрываем соединение с rootDB (получаем *sql.DB)
	sqlDB, err := rootDB.DB()
	if err == nil {
		sqlDB.Close()
	}

	// 2. Подключаемся к целевой базе данных
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to application database: %v", err)
	}

	// 3. Автомиграция
	if err := db.AutoMigrate(&entities.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
