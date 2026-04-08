package spotrecommender

import (
	"fmt"
	"surf-recommender/pkg/telegram"
	"time"
)

// RunRecommend получает рекомендацию от Claude и отправляет её в Telegram-канал.
// Если dryRun=true — выводит результат в stdout, в Telegram не отправляет.
func RunRecommend(svc *Service, tg *telegram.Client, chatID, spot string, date time.Time, dryRun bool) error {
	result, err := svc.Recommend(spot, date)
	if err != nil {
		return fmt.Errorf("recommendation failed: %w", err)
	}

	if dryRun {
		fmt.Println(result)
		return nil
	}

	if err := tg.SendMessage(chatID, result); err != nil {
		return fmt.Errorf("send to telegram: %w", err)
	}

	fmt.Printf("✓ Рекомендация отправлена в %s\n", chatID)
	return nil
}
