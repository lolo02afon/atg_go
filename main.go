package main

import (
	"atg_go/internal/auth"
	"atg_go/internal/comments"
	"atg_go/pkg/storage"
	"database/sql"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Инициализация подключения к БД
	dbConn, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/atg_db?sslmode=disable")
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

	// Группа роутов для авторизации
	authGroup := r.Group("/auth")
	auth.SetupRoutes(authGroup, db) // Передаем только хранилище аккаунтов

	// Группа роутов для комментариев
	commentGroup := r.Group("/comment")
	comments.SetupRoutes(commentGroup, db, commentDB) // Передаем оба хранилища

	// Группа роутов для реакций на чужие комментарии
	// reactionGroup := r.Group("/reaction")
	// reaction.SetupRoutes(reactionGroup, db, commentDB)

	// Группа роутов для telegram-модуля
	// telegramGroup := r.Group("/module")
	// telegramModule.SetupRoutes(telegramGroup)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Логирование зарегистрированных роутов
	log.Printf("[ROUTER] Routes initialized:")
	log.Printf("[ROUTER] POST /auth/CreateAccount")
	log.Printf("[ROUTER] POST /comment/send")
	log.Printf("[ROUTER] GET /health")

	return r
}
