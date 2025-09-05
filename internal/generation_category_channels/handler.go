package generation_category_channels

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"atg_go/internal/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	"atg_go/pkg/telegram"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы генерации подборки каналов.
type Handler struct {
	DB *storage.DB
}

// NewHandler создаёт новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{DB: db}
}

type request struct {
	NameCategory     string   `json:"name_category" binding:"required"`
	InputChannels    []string `json:"input_channels" binding:"required"`
	ResultCountLinks int      `json:"result_count_links" binding:"required"`
}

// GenerateCategory обрабатывает POST-запрос и формирует новую категорию каналов.
func (h *Handler) GenerateCategory(c *gin.Context) {
	var req request
	if err := c.ShouldBindJSON(&req); err != nil {
		// Логируем проблему с распознаванием входных данных
		log.Printf("[GENERATION ERROR] некорректный формат запроса: %v", err)
		httputil.RespondError(c, http.StatusBadRequest, "invalid request format")
		return
	}

	accounts, err := h.DB.GetMonitoringAccounts()
	if err != nil {
		// Фиксируем ошибку получения аккаунтов мониторинга
		log.Printf("[GENERATION ERROR] не удалось получить аккаунты мониторинга: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "failed to get monitoring accounts")
		return
	}
	if len(accounts) == 0 {
		// Сообщаем о пустом списке аккаунтов
		log.Printf("[GENERATION ERROR] аккаунты мониторинга не найдены")
		httputil.RespondError(c, http.StatusNotFound, "monitoring accounts not found")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	results := make(map[string]struct{})

	queues := make([][]string, len(accounts))
	for i, ch := range req.InputChannels {
		queues[i%len(accounts)] = append(queues[i%len(accounts)], ch)
	}

	var wg sync.WaitGroup
	for i, acc := range accounts {
		queue := append([]string(nil), queues[i]...)
		wg.Add(1)
		go func(account models.Account, q []string) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			processed := make(map[string]struct{})
			for len(q) > 0 {
				select {
				case <-ctx.Done():
					return
				default:
				}
				url := q[0]
				q = q[1:]
				if _, ok := processed[url]; ok {
					continue
				}
				processed[url] = struct{}{}
				recs, err := telegram.GetChannelRecommendations(h.DB, account, url)
				if err != nil {
					// Предупреждение при проблемах с получением рекомендаций по каналу
					log.Printf("[GENERATION WARN] не удалось получить рекомендации для %s: %v", url, err)
					continue
				}
				mu.Lock()
				for _, link := range recs {
					if _, exists := results[link]; !exists {
						results[link] = struct{}{}
						// Каждые десять найденных каналов фиксируем в логах
						if len(results)%10 == 0 {
							log.Printf("[GENERATION INFO] записано %d похожих каналов, последний: %s", len(results), link)
						}
						if len(results) >= req.ResultCountLinks {
							mu.Unlock()
							cancel()
							return
						}
						q = append(q, link)
					}
				}
				mu.Unlock()
				time.Sleep(time.Duration(500+r.Intn(1000)) * time.Millisecond)
			}
		}(acc, queue)
	}

	wg.Wait()

	mu.Lock()
	urls := make([]string, 0, len(results))
	for url := range results {
		urls = append(urls, url)
	}
	mu.Unlock()

	if _, err := h.DB.CreateCategory(req.NameCategory, urls); err != nil {
		// Логируем ошибку сохранения итоговой категории
		log.Printf("[GENERATION ERROR] не удалось сохранить категорию: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "failed to save category")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"count":  len(urls),
	})
}
