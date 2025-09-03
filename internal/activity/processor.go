package activity

import (
	"atg_go/internal/common"
	"atg_go/internal/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProcessAccounts выполняет общие шаги массовых операций над аккаунтами.
// Выносим задержки и выбор каналов сюда, чтобы не дублировать код в хэндлерах.
// Коллбэк отвечает только за конкретное действие (комментарий, реакция) и
// сообщает, было ли оно успешным.
func ProcessAccounts(
	c *gin.Context,
	accounts []models.Account,
	commentDB *storage.CommentDB,
	send func(models.Account, string) (bool, error),
) (successCount, errorCount int, err error) {
	// Инициализируем генератор случайных чисел один раз, чтобы паузы и
	// выбор каналов не повторяли один и тот же шаблон.
	rand.Seed(time.Now().UnixNano())

	// Максимальное количество попыток для одного аккаунта.
	const maxAttempts = 10

	for i, account := range accounts {
		if i > 0 {
			// Пауза между аккаунтами делает активность менее подозрительной.
			log.Printf("[HANDLER] Аккаунт %s обработан. Ожидание перед следующим...", accounts[i-1].Phone)
			if err := common.WaitWithCancellation(c.Request.Context(), [2]int{6, 15}); err != nil {
				// При отмене запроса прекращаем работу, чтобы не тратить ресурсы зря.
				log.Printf("[HANDLER WARN] Запрос отменён во время ожидания: %v", err)
				c.JSON(http.StatusRequestTimeout, gin.H{
					"status":     "Cancelled during delay",
					"processed":  i,
					"successful": successCount,
					"failed":     errorCount,
				})
				return successCount, errorCount, err
			}
		}

		// Несколько попыток выполнить действие: пока не удастся или не исчерпаем лимит.
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			// Выбираем канал для текущей попытки. Отсутствие каналов считаем фатальной ошибкой.
			channelURL, err := storage.PickRandomChannel(commentDB, *account.OrderID)
			if err != nil {
				if errors.Is(err, storage.ErrNoChannel) {
					log.Printf("[HANDLER ERROR] Нет доступных каналов: %v", err)
					httputil.RespondError(c, http.StatusNotFound, "No channels available")
					return successCount, errorCount, err
				}
				// Прочие ошибки фиксируем и пробуем ещё раз.
				log.Printf("[HANDLER WARN] Ошибка выбора канала для %s (попытка %d): %v", account.Phone, attempt, err)
				if attempt == maxAttempts {
					errorCount++
				}
				continue
			}
			log.Printf("[HANDLER INFO] Выбран канал для %s: %s (попытка %d)", account.Phone, channelURL, attempt)

			// Коллбэк отправляет действие и возвращает, было ли оно успешным.
			ok, err := send(account, channelURL)
			if err != nil {
				log.Printf("[HANDLER WARN] Ошибка обработки для %s (попытка %d): %v", account.Phone, attempt, err)
				if attempt == maxAttempts {
					errorCount++
				}
				continue
			}
			if ok {
				successCount++
			} else if attempt == maxAttempts {
				// Все попытки исчерпаны, результат не получен.
				errorCount++
			}
			break
		}
	}

	return successCount, errorCount, nil
}
