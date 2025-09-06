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
	accountmutex "atg_go/pkg/telegram/module/account_mutex"

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

	accounts, err := h.DB.GetGeneratorCategoryAccounts()
	if err != nil {
		// Фиксируем ошибку получения аккаунтов генерации
		log.Printf("[GENERATION ERROR] не удалось получить аккаунты генерации категорий: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "failed to get generator accounts")
		return
	}
	if len(accounts) == 0 {
		// Сообщаем о пустом списке аккаунтов
		log.Printf("[GENERATION ERROR] аккаунты генерации категорий не найдены")
		httputil.RespondError(c, http.StatusNotFound, "generator accounts not found")
		return
	}
	log.Printf("[GENERATION DEBUG] получено %d аккаунтов генерации: %v", len(accounts), accountIDs(accounts))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Отбираем только свободные аккаунты и сразу блокируем их,
	// чтобы избежать параллельного использования.
	var (
		free []models.Account
		busy []int
	)
	for _, acc := range accounts {
		if err := accountmutex.LockAccount(acc.ID); err != nil {
			log.Printf("[GENERATION WARN] аккаунт %d пропущен: %v", acc.ID, err)
			busy = append(busy, acc.ID)
			continue
		}
		free = append(free, acc)
	}
	log.Printf("[GENERATION DEBUG] свободные аккаунты: %v, занятые: %v", accountIDs(free), busy)
	if len(free) == 0 {
		log.Printf("[GENERATION ERROR] нет свободных аккаунтов для генерации, заняты: %v", busy)
		httputil.RespondError(c, http.StatusServiceUnavailable, "no free accounts")
		return
	}

	var mu sync.Mutex
	// results хранит уникальные найденные ссылки в порядке появления
	results := make([]string, 0, req.ResultCountLinks)
	// seen используется для отслеживания уже добавленных ссылок
	seen := make(map[string]struct{})

	queues := make([][]string, len(free))
	for i, ch := range req.InputChannels {
		queues[i%len(free)] = append(queues[i%len(free)], ch)
	}

	var wg sync.WaitGroup
	for i, acc := range free {
		queue := append([]string(nil), queues[i]...)
		wg.Add(1)
		go func(account models.Account, q []string) {
			defer wg.Done()
			defer accountmutex.UnlockAccount(account.ID)
			log.Printf("[GENERATION DEBUG] аккаунт %d начал обработку %d каналов", account.ID, len(q))
			defer log.Printf("[GENERATION DEBUG] аккаунт %d завершил обработку", account.ID)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			processed := make(map[string]struct{})
			for len(q) > 0 {
				select {
				case <-ctx.Done():
					log.Printf("[GENERATION DEBUG] аккаунт %d остановлен по сигналу контекста", account.ID)
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
				for _, link := range recs {
					ok, err := telegram.HasAccessibleDiscussion(h.DB, account, link)
					if err != nil {
						// Сообщаем об ошибке проверки обсуждения
						log.Printf("[GENERATION WARN] не удалось проверить обсуждение для %s: %v", link, err)
						continue
					}
					if !ok {
						// Пропускаем канал без свободного обсуждения
						continue
					}

					mu.Lock()
					if _, exists := seen[link]; exists {
						mu.Unlock()
						continue
					}
					seen[link] = struct{}{}
					results = append(results, link)
					// Каждые десять найденных каналов фиксируем в логах
					if len(results)%10 == 0 {
						log.Printf("[GENERATION INFO] записано %d похожих каналов, последний: %s", len(results), link)
					}
					if len(results) >= req.ResultCountLinks {
						log.Printf("[GENERATION DEBUG] достигнут лимит %d результатов", req.ResultCountLinks)
						mu.Unlock()
						cancel()
						return
					}
					mu.Unlock()

					q = append(q, link)
				}
				time.Sleep(time.Duration(500+r.Intn(1000)) * time.Millisecond)
			}
		}(acc, queue)
	}

	wg.Wait()

	mu.Lock()
	// Копируем найденные ссылки, чтобы вернуть их вызывающему коду
	urls := append([]string(nil), results...)
	mu.Unlock()
	log.Printf("[GENERATION INFO] найдено %d ссылок для категории %s", len(urls), req.NameCategory)

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

// accountIDs возвращает идентификаторы аккаунтов для вывода в журнал.
func accountIDs(accs []models.Account) []int {
	ids := make([]int, 0, len(accs))
	for _, a := range accs {
		ids = append(ids, a.ID)
	}
	return ids
}
