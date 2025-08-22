package common

import (
	"context"
	"math/rand"
	"time"
)

// WaitWithCancellation выполняет ожидание в случайном диапазоне и
// регулярно проверяет контекст на отмену, чтобы не блокировать долгие задержки.
// Используем шаг в пять секунд, чтобы можно было вовремя завершить работу по требованию.
func WaitWithCancellation(ctx context.Context, delayRange [2]int) error {
	delay := rand.Intn(delayRange[1]-delayRange[0]+1) + delayRange[0]
	for remaining := delay; remaining > 0; {
		step := 5
		if remaining < step {
			step = remaining
		}
		select {
		case <-ctx.Done():
			// Возвращаем ошибку контекста, чтобы вызвать обработку прерывания выше по стеку.
			return ctx.Err()
		case <-time.After(time.Duration(step) * time.Second):
		}
		remaining -= step
	}
	return nil
}
