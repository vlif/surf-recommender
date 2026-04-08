package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"surf-recommender/internal/app"
	internalconfig "surf-recommender/internal/config"
	"surf-recommender/internal/regions"
	spotrecommender "surf-recommender/internal/spot-recommender"
	"surf-recommender/pkg/anthropic"
	"surf-recommender/pkg/stormglass"
	"surf-recommender/pkg/telegram"
)

var registry map[string]regions.Region

func main() {
	_ = godotenv.Load()

	var err error
	registry, err = internalconfig.LoadAllRegions("config/regions")
	if err != nil {
		log.Fatalf("Загрузка регионов: %v", err)
	}

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
	cronSchedule := os.Getenv("CRON_SCHEDULE")
	if cronSchedule == "" {
		cronSchedule = "0 6 * * *" // 06:00 по часовому поясу CRON_TIMEZONE
	}
	cronTimezone := os.Getenv("CRON_TIMEZONE")
	if cronTimezone == "" {
		cronTimezone = "Europe/Lisbon"
	}

	regionFlag := flag.String("region", "algarve", fmt.Sprintf("Регион сёрфинга %v", availableRegions()))
	spot := flag.String("spot", "", "Конкретный спот для анализа (например: Burgau, Arrifana)")
	dateStr := flag.String("date", "", "Дата прогноза YYYY-MM-DD (по умолчанию: сегодня, только для ручного запуска)")
	daemon := flag.Bool("daemon", false, "Запустить как планировщик (по расписанию CRON_SCHEDULE)")
	mockMode := flag.Bool("mock", false, "Тестовый режим: данные из файла вместо Stormglass API")
	mockFile := flag.String("mock-file", "testdata/stormglass_response.json", "Путь к JSON-файлу с тестовыми данными")
	flag.Parse()

	region, ok := registry[*regionFlag]
	if !ok {
		log.Fatalf("Неизвестный регион: %q. Доступные: %v", *regionFlag, availableRegions())
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

	if *daemon {
		runDaemon(svc, tgClient, telegramChatID, *spot, cronSchedule, cronTimezone)
		return
	}

	// Ручной запуск
	date := time.Now()
	if *dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", *dateStr)
		if err != nil {
			log.Fatalf("Неверный формат даты: %s (используй YYYY-MM-DD)", *dateStr)
		}
	}

	if err := spotrecommender.RunRecommend(svc, tgClient, telegramChatID, *spot, date, *mockMode); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon(svc *spotrecommender.Service, tg *telegram.Client, chatID, spot, cronSchedule, cronTimezone string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := app.New(svc, tg, chatID, spot).Start(ctx, cronSchedule, cronTimezone); err != nil {
		log.Fatalf("Ошибка планировщика: %v", err)
	}
}

func availableRegions() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}
