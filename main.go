package main

import (
	"atg_go/internal/auth"
	"atg_go/internal/comments"
	"atg_go/internal/middleware"
	module "atg_go/internal/module"
	orders "atg_go/internal/order"
	reaction "atg_go/internal/reaction"
	statistics "atg_go/internal/statistics"
	telegram "atg_go/internal/telegram"
	"atg_go/pkg/storage"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Set default timezone to Moscow (MSK)
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalf("Failed to load location: %v", err)
	}
	time.Local = loc
	_ = os.Setenv("TZ", "Europe/Moscow")

	// Инициализация подключения к БД
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// запасной вариант для локального запуска разработчиком
		dsn = "postgres://postgres:postgres@localhost:5432/atg_db?sslmode=disable&application_name=atg_app"
	}
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Проверка подключения
	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	// Инициализация хранилищ
	db := storage.NewDB(dbConn)               // Для работы с аккаунтами
	commentDB := storage.NewCommentDB(dbConn) // Для работы с каналами

	// Запускаем фоновые процессы Telegram один раз
	telegram.Run(db)

	// Настройка роутера
	r := setupRouter(db, commentDB)

	// Запуск сервера
	port := getPort()
	log.Printf("Starting server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Функция получения порта из переменных окружения
func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}

// Настройка маршрутов
func setupRouter(db *storage.DB, commentDB *storage.CommentDB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.AuthRequired())

	// Группа роутов для авторизации
	authGroup := r.Group("/auth")
	auth.SetupRoutes(authGroup, db) // Передаем только хранилище аккаунтов

	// Группа роутов для комментариев
	commentGroup := r.Group("/comment")
	comments.SetupRoutes(commentGroup, db, commentDB) // Передаем оба хранилища

	// Группа роутов для реакций на чужие комментарии
	reactionGroup := r.Group("/reaction")
	reaction.SetupRoutes(reactionGroup, db, commentDB)

	// Группа роутов для telegram-модуля
	moduleGroup := r.Group("/module")
	module.SetupRoutes(moduleGroup, db)

	// Группа роутов для заказов
	orderGroup := r.Group("/order")
	orders.SetupRoutes(orderGroup, db)

	// Группа роутов для статистики
	statsGroup := r.Group("/statistics")
	statistics.SetupRoutes(statsGroup, db)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
