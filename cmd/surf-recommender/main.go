package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	spotrecommender "surf-recommender/internal/spot-recommender"
	"surf-recommender/internal/regions"
	"surf-recommender/internal/regions/algarve"
	"surf-recommender/pkg/anthropic"
	"surf-recommender/pkg/stormglass"
	"surf-recommender/pkg/telegram"
)

// registry — реестр доступных регионов.
// Чтобы добавить новый регион: создай пакет internal/regions/<name>
// и добавь его сюда одной строкой.
var registry = map[string]regions.Region{
	"algarve": algarve.Region,
	// "lisbon": lisbon.Region,  // будет добавлено позже
}

func main() {
	_ = godotenv.Load()

	stormglassToken := os.Getenv("STORMGLASS_TOKEN")
	if stormglassToken == "" {
		log.Fatal("STORMGLASS_TOKEN не задан")
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		log.Fatal("ANTHROPIC_API_KEY не задан")
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан")
	}

	anthropicModel := os.Getenv("ANTHROPIC_MODEL")
	if anthropicModel == "" {
		anthropicModel = anthropic.DefaultModel
	}

	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
	if telegramChatID == "" {
		telegramChatID = "@surfing_in_portugal"
	}

	regionFlag := flag.String("region", "algarve", fmt.Sprintf("Регион сёрфинга %v", availableRegions()))
	spot := flag.String("spot", "", "Конкретный спот для анализа (например: Burgau, Arrifana)")
	dateStr := flag.String("date", "", "Дата прогноза YYYY-MM-DD (по умолчанию: сегодня)")
	mockMode := flag.Bool("mock", false, "Тестовый режим: данные из файла вместо Stormglass API (не тратит квоту)")
	mockFile := flag.String("mock-file", "testdata/stormglass_response.json", "Путь к JSON-файлу с тестовыми данными")
	flag.Parse()

	region, ok := registry[*regionFlag]
	if !ok {
		log.Fatalf("Неизвестный регион: %q. Доступные: %v", *regionFlag, availableRegions())
	}

	date := time.Now()
	if *dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", *dateStr)
		if err != nil {
			log.Fatalf("Неверный формат даты: %s (используй YYYY-MM-DD)", *dateStr)
		}
	}

	var sgClient stormglass.Fetcher
	if *mockMode {
		log.Printf("[mock] данные из файла: %s", *mockFile)
		sgClient = stormglass.NewMockClient(*mockFile)
	} else {
		sgClient = stormglass.NewClient(stormglassToken)
	}
	anthropicClient := anthropic.NewClient(anthropicKey, anthropicModel)
	tgClient := telegram.NewClient(telegramToken)
	svc := spotrecommender.NewService(sgClient, anthropicClient, region)

	if err := spotrecommender.RunRecommend(svc, tgClient, telegramChatID, *spot, date); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func availableRegions() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}
