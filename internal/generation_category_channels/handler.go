package generation_category_channels

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"atg_go/internal/technical/httputil"
	"atg_go/models"
	"atg_go/pkg/storage"
	gcc "atg_go/pkg/telegram/generation_category_channels"
	accountmutex "atg_go/pkg/telegram/technical/account_mutex"

	"github.com/gin-gonic/gin"
)

// Handler обрабатывает запросы генерации подборки каналов.
type Handler struct {
	DB    *storage.DB
	queue chan struct{} // очередь на выполнение запросов генерации
}

// NewHandler создаёт новый экземпляр обработчика.
func NewHandler(db *storage.DB) *Handler {
	return &Handler{
		DB:    db,
		queue: make(chan struct{}, 1), // допускаем единовременную обработку одного запроса
	}
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

	// Ставим запрос в очередь: пока предыдущий не завершён, новый ожидает
	h.queue <- struct{}{}
	defer func() { <-h.queue }()

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

	// results хранит уникальные найденные ссылки в порядке появления
	results := make([]string, 0, req.ResultCountLinks)
	// seen используется для отслеживания уже добавленных ссылок
	seen := make(map[string]struct{})
	// processed помогает не запрашивать рекомендации повторно для одного и того же канала
	processed := make(map[string]struct{})

	// очередь каналов на обработку
	queue := append([]string(nil), req.InputChannels...)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	accIdx := 0 // индекс аккаунта для последовательного использования

	// Освобождаем занятые аккаунты по завершении работы
	defer func() {
		for _, a := range free {
			accountmutex.UnlockAccount(a.ID)
		}
	}()

	// Сначала проверяем входные каналы и добавляем подходящие в результаты
	for _, link := range req.InputChannels {
		if len(results) >= req.ResultCountLinks {
			break
		}
		acc := free[accIdx%len(free)]
		accIdx++
		h.appendChannelIfAccessible(acc, link, seen, &results)
	}

	for len(queue) > 0 && len(results) < req.ResultCountLinks {
		url := queue[0]
		queue = queue[1:]
		if _, ok := processed[url]; ok {
			continue
		}
		processed[url] = struct{}{}

		acc := free[accIdx%len(free)]
		accIdx++
		recs, err := gcc.GetChannelRecommendations(h.DB, acc, url)
		if err != nil {
			log.Printf("[GENERATION WARN] не удалось получить рекомендации для %s аккаунтом %d: %v", url, acc.ID, err)
			continue
		}

		for _, link := range recs {
			accCheck := free[accIdx%len(free)]
			accIdx++
			if !h.appendChannelIfAccessible(accCheck, link, seen, &results) {
				continue
			}
			if len(results)%10 == 0 {
				log.Printf("[GENERATION INFO] записано %d похожих каналов, последний: %s", len(results), link)
			}
			if len(results) >= req.ResultCountLinks {
				break
			}
			queue = append(queue, link)
			time.Sleep(time.Duration(500+r.Intn(1000)) * time.Millisecond)
		}

		if len(results) >= req.ResultCountLinks {
			break
		}
	}

	log.Printf("[GENERATION INFO] найдено %d ссылок для категории %s", len(results), req.NameCategory)

	if _, err := h.DB.CreateCategory(req.NameCategory, results); err != nil {
		// Логируем ошибку сохранения итоговой категории
		log.Printf("[GENERATION ERROR] не удалось сохранить категорию: %v", err)
		httputil.RespondError(c, http.StatusInternalServerError, "failed to save category")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"count":  len(results),
	})
}

// appendChannelIfAccessible проверяет, что у канала есть открытое обсуждение,
// и добавляет ссылку в результаты, если её ещё не было.
// Возвращает true, если ссылка добавлена.
func (h *Handler) appendChannelIfAccessible(acc models.Account, link string, seen map[string]struct{}, results *[]string) bool {
	ok, err := gcc.HasAccessibleDiscussion(h.DB, acc, link)
	if err != nil {
		log.Printf("[GENERATION WARN] не удалось проверить обсуждение для %s аккаунтом %d: %v", link, acc.ID, err)
		return false
	}
	if !ok {
		return false
	}
	if _, exists := seen[link]; exists {
		return false
	}
	seen[link] = struct{}{}
	*results = append(*results, link)
	return true
}

// accountIDs возвращает идентификаторы аккаунтов для вывода в журнал.
func accountIDs(accs []models.Account) []int {
	ids := make([]int, 0, len(accs))
	for _, a := range accs {
		ids = append(ids, a.ID)
	}
	return ids
}
