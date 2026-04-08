# surf-recommender

CLI-инструмент для рекомендаций спотов для сёрфинга на основе актуального прогноза погоды.

Получает почасовые данные по волнам, свеллу, ветру и приливам из **Stormglass API**, передаёт их в **Claude (Anthropic)** вместе с базой спотов и правилами интерпретации — и получает рекомендацию на русском языке. Результат отправляется в **Telegram-канал**.

Поддерживаемые регионы: **Алгарве** (база Лагуш) и **Лиссабон**. Архитектура позволяет добавлять новые регионы без изменения кода.

---

## Установка

```bash
git clone <repo>
cd surf-recommender
go build -o surf-recommender ./cmd/surf-recommender
```

## Конфигурация

```bash
cp .env.example .env
```

```env
STORMGLASS_TOKEN=your_stormglass_token      # обязательно
ANTHROPIC_API_KEY=your_anthropic_api_key    # обязательно
TELEGRAM_BOT_TOKEN=your_telegram_bot_token  # обязательно

TELEGRAM_CHAT_ID=@surfing_in_portugal       # опционально, это значение по умолчанию
ANTHROPIC_MODEL=claude-sonnet-4-6           # опционально

# Планировщик (только для --daemon режима)
CRON_SCHEDULE=0 6 * * *                     # опционально, по умолчанию 06:00
CRON_TIMEZONE=Europe/Lisbon                 # опционально, по умолчанию Europe/Lisbon
```

---

## Команды

**Рекомендация на сегодня:**
```bash
./surf-recommender                        # Алгарве (по умолчанию)
./surf-recommender --region lisbon        # Лиссабон
```

**Рекомендация для конкретного спота:**
```bash
./surf-recommender --spot Burgau
./surf-recommender --region lisbon --spot Carcavelos
```

**Прогноз на конкретную дату:**
```bash
./surf-recommender --date 2026-04-10
./surf-recommender --date 2026-04-10 --spot Zavial
```

**Запуск по расписанию** (daemon-режим):
```bash
./surf-recommender --daemon
```
Блокируется и отправляет рекомендацию каждый день в 06:00 по Лиссабону. Останавливается по Ctrl+C или SIGTERM.

```bash
CRON_SCHEDULE="0 7 * * *" CRON_TIMEZONE="Europe/Lisbon" ./surf-recommender --daemon
```

**Тестовый режим** (не тратит квоту Stormglass, выводит в stdout):
```bash
./surf-recommender --mock
./surf-recommender --mock --region lisbon
./surf-recommender --mock --mock-file path/to/custom_data.json
```

---

## Тестовый режим

Stormglass API имеет ограничение на количество запросов в день. Флаг `--mock` позволяет запускать инструмент без обращения к Stormglass — данные берутся из локальных JSON-файлов. В mock-режиме результат выводится в stdout, в Telegram ничего не отправляется.

Файлы по умолчанию:
- `testdata/stormglass_response.json` — прогноз волн и ветра
- `testdata/stormglass_tides.json` — приливные экстремумы

Даты в файлах автоматически сдвигаются на сегодня/завтра, поэтому можно использовать одни и те же файлы многократно.

Fixture содержит реалистичный сценарий:
- **Утро (06:00–11:00 UTC)**: северный ветер ~8 км/ч (оффшор), свелл 0.8–0.9м@10с — хорошее окно
- **День (12:00–18:00 UTC)**: ветер поворачивает на юго-запад (оншор)
- **Приливы**: максимум в 04:15, минимум в 10:30 — убывающий прилив в утреннее окно

---

## Деплой через GitHub Actions

Два workflow в `.github/workflows/`:

**`release.yml`** — собирает бинарник и Docker-образ при создании тега:
```bash
git tag v1.0.0 && git push --tags
```
Образ публикуется в `ghcr.io/vlif/surf-recommender`.

**`daily.yml`** — запускает рекомендацию для обоих регионов каждый день в 06:00 UTC (07:00 по Лиссабону летом). Регионы запускаются параллельно. Можно запустить вручную через GitHub → Actions → Daily Surf Recommendation → Run workflow.

Необходимые секреты в настройках репозитория (`Settings → Secrets → Actions`):

| Секрет | Описание |
|---|---|
| `STORMGLASS_TOKEN` | Токен Stormglass API |
| `ANTHROPIC_API_KEY` | Ключ Anthropic API |
| `TELEGRAM_BOT_TOKEN` | Токен Telegram-бота |
| `TELEGRAM_CHAT_ID` | ID канала, например `@surfing_in_portugal` |

Запуск через Docker вручную:
```bash
docker run --env-file .env ghcr.io/vlif/surf-recommender:latest
docker run --env-file .env ghcr.io/vlif/surf-recommender:latest --region lisbon
```

---

## Архитектура

```
cmd/surf-recommender/main.go        # CLI: флаги --region --spot --date --mock --daemon
internal/
  regions/
    region.go                       # типы Region, ForecastPoint, TidePoint
  config/
    loader.go                       # загрузка регионов из config/regions/
  spot-recommender/
    service.go                      # бизнес-логика: fetch → Claude → результат
    commands.go                     # вывод результата (Telegram или stdout)
  app/
    app.go                          # планировщик (gocron), graceful shutdown
pkg/
  stormglass/
    fetcher.go                      # интерфейсы Fetcher и TideFetcher
    client.go                       # реальный Stormglass API клиент (прогноз + приливы)
    mock.go                         # mock-клиенты из JSON-файлов
  anthropic/client.go               # Anthropic Messages API клиент
  telegram/client.go                # Telegram Bot API клиент
config/
  regions/
    base.txt                        # общие правила для всех регионов (компас, свелл, приливы)
    algarve/
      config.json                   # id, display_name, forecast_points, tide_point
      prompt.txt                    # региональный промпт для Claude
    lisbon/
      config.json
      prompt.txt
testdata/
  stormglass_response.json          # fixture прогноза для тестового режима
  stormglass_tides.json             # fixture приливов для тестового режима
```

### Как работает

1. Stormglass возвращает почасовые данные по точкам побережья (06:00–18:00 UTC) и приливные экстремумы за два дня
2. Claude получает все часы по порядку, данные о приливах и сам находит лучшее временное окно для сессии
3. Рекомендация на русском с ссылками на Google Maps отправляется в Telegram-канал

### Структура промптов

Системный промпт для каждого региона состоит из двух частей:
- `config/regions/<region>/prompt.txt` — специфика региона: споты, правила ветра, направления свелла, формат ответа
- `config/regions/base.txt` — общие правила: обозначения сторон света, пороги скорости ветра, размер и период свелла, логика приливов

Loader автоматически объединяет их при старте.

### Почему используется поле `.sg` из ответа Stormglass

Stormglass возвращает значения каждого параметра сразу из нескольких погодных моделей:

```json
"waveHeight": { "sg": 0.82, "noaa": 0.75, "icon": 0.88 }
```

`sg` — собственная сводная модель Stormglass, которая блендирует несколько источников в одно значение. Мы берём именно её, потому что она всегда присутствует — остальные источники могут отсутствовать для конкретной точки или параметра.

---

## Добавить новый регион

1. Создай директорию `config/regions/<name>/`
2. Добавь `config.json`:

```json
{
  "id": "porto",
  "display_name": "Порту",
  "tide_point": { "lat": 41.15, "lng": -8.68 },
  "forecast_points": [
    { "name": "Побережье Порту", "lat": 41.15, "lng": -8.68 }
  ]
}
```

3. Добавь `prompt.txt` — описание спотов, правила ветра и свелла для региона, формат ответа. Общие правила (компас, пороги ветра, свелл) подхватятся автоматически из `base.txt`.

Регион появится при следующем запуске — код менять не нужно.

---

## Споты Алгарве (база — Лагуш)

| Спот | Дорога | Побережье |
|---|---|---|
| Meia Praia | 5 мин | южное |
| Luz | 15 мин | южное |
| Burgau | 20 мин | южное |
| Salema | 25 мин | южное |
| Zavial | 30 мин | южное |
| Mareta | 35 мин | южное |
| Tonel | 35 мин | западное |
| Castelejo | 40 мин | западное |
| Arrifana | 40 мин | западное |
| Amado | 45 мин | западное |

## Споты Лиссабон (база — Лиссабон)

| Спот | Дорога | Побережье |
|---|---|---|
| Caparica | 25 мин | Капарика |
| Carcavelos | 30 мин | Кашкайш |
| São Julião | 35 мин | Кашкайш |
| Guincho | 45 мин | Кашкайш |
| Praia Grande | 55 мин | Синтра |
