package main

import (
	"atg_go/handlers"
	"atg_go/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	// строка подключения к PostgreSQL
	dsn := "postgres://postgres:postgres@localhost:5432/atg_db?sslmode=disable"

	// создаём подключение к БД
	db := storage.NewDB(dsn)

	// создаём хендлер для аккаунтов
	accountHandler := handlers.NewAccountHandler(db)

	verificationHandler := handlers.NewVerificationHandler(db)

	// инициализируем Gin
	r := gin.Default()

	// маршрут для создания аккаунта
	r.POST("/account", accountHandler.CreateAccount)

	r.POST("/verifications", verificationHandler.Create)
	r.PUT("/verifications", verificationHandler.Update)

	// запуск сервера
	r.Run(":8080")
}
