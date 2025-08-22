package httputil

import "github.com/gin-gonic/gin"

// RespondError отправляет сообщение об ошибке в едином формате и прекращает обработку запроса.
// Используем AbortWithStatusJSON, чтобы последующие обработчики не выполнялись, даже если забыли вернуть управление.
func RespondError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
