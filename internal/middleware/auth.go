package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const bearerToken = "ZXNzIiwiZXhwIjoxNzUyOTU3OTMyLCJpYXQiOjE3NTI5NTQzMzIsImp0aSI6ImM1ZjY0MjcwMjZjYjY1IiwidXNlcl9pZRcNAW-s02Ayz6A"

// AuthRequired проверяет наличие корректного статичного Bearer-токена
func AuthRequired() gin.HandlerFunc {
	expected := "Bearer " + bearerToken
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != expected {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
