package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	spotrecommender "surf-recommender/internal/spot-recommender"
	"surf-recommender/pkg/telegram"
)

// App запускает планировщик и выполняет задание по расписанию.
type App struct {
	svc    *spotrecommender.Service
	tg     *telegram.Client
	chatID string
	spot   string
}

func New(svc *spotrecommender.Service, tg *telegram.Client, chatID, spot string) *App {
	return &App{svc: svc, tg: tg, chatID: chatID, spot: spot}
}


// Start запускает планировщик с заданным крон-выражением и часовым поясом.
// Блокируется до отмены ctx, затем выполняет graceful shutdown.
func (a *App) Start(ctx context.Context, cronExpr, timezone string) error {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("неверный часовой пояс %q: %w", timezone, err)
	}

	sch, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return fmt.Errorf("создание планировщика: %w", err)
	}

	if _, err := sch.NewJob(
		gocron.CronJob(cronExpr, false),
		gocron.NewTask(a.job),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	); err != nil {
		return fmt.Errorf("регистрация задания: %w", err)
	}

	sch.Start()
	log.Printf("Планировщик запущен. Расписание: %q, часовой пояс: %s", cronExpr, timezone)

	<-ctx.Done()
	log.Println("Получен сигнал завершения, останавливаю планировщик...")

	if err := sch.Shutdown(); err != nil {
		return fmt.Errorf("остановка планировщика: %w", err)
	}
	return nil
}

// job выполняется по расписанию.
func (a *App) job() {
	log.Println("Запуск задания по расписанию...")
	if err := spotrecommender.RunRecommend(a.svc, a.tg, a.chatID, a.spot, time.Now(), false); err != nil {
		log.Printf("Ошибка выполнения задания: %v", err)
	}
}
