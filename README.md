# surf-recommender

CLI-инструмент для рекомендаций спотов для сёрфинга на основе актуального прогноза погоды.

Получает почасовые данные по волнам, зыби и ветру из **Stormglass API**, передаёт их в **Claude (Anthropic)** вместе с базой спотов и правилами интерпретации — и получает рекомендацию на русском языке. Результат отправляется в **Telegram-канал**.

Сейчас поддерживается регион **Алгарве** (западный Алгарве, база Лагуш). Архитектура позволяет добавлять новые регионы без изменения сервиса.

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

**Рекомендация на сегодня** (лучший спот в регионе):
```bash
./surf-recommender
```

**Рекомендация для конкретного спота:**
```bash
./surf-recommender --spot Burgau
./surf-recommender --spot Arrifana
```

**Прогноз на конкретную дату:**
```bash
./surf-recommender --date 2026-04-10
./surf-recommender --date 2026-04-10 --spot Zavial
```

**Выбор региона:**
```bash
./surf-recommender --region algarve   # по умолчанию
```

**Запуск по расписанию** (daemon-режим):
```bash
./surf-recommender --daemon
```
Блокируется и отправляет рекомендацию каждый день в 06:00 по Лиссабону. Останавливается по Ctrl+C или SIGTERM.

Изменить расписание или часовой пояс через env:
```bash
CRON_SCHEDULE="0 7 * * *" CRON_TIMEZONE="Europe/Lisbon" ./surf-recommender --daemon
```

**Тестовый режим** (не тратит квоту Stormglass):
```bash
./surf-recommender --mock
./surf-recommender --mock --mock-file path/to/custom_data.json
```

---

## Тестовый режим

Stormglass API имеет ограничение на количество запросов в день. Флаг `--mock` позволяет запускать инструмент без обращения к API — данные берутся из локального JSON-файла.

Файл по умолчанию: `testdata/stormglass_response.json`

Формат файла совпадает с ответом Stormglass API. Даты в файле автоматически сдвигаются на сегодня/завтра, поэтому можно использовать один файл многократно.

Fixture содержит реалистичный сценарий:
- **Утро (06:00–11:00 UTC)**: N ветер ~8 км/ч (оффшор), зыбь 0.8–0.9м@10с — хорошее окно
- **День (12:00–18:00 UTC)**: ветер поворачивает на SW (оншор) — Claude определяет лучшее окно сессии

---

## Пример вывода в Telegram

```
🏆 ТОП ВЫБОР: Meia Praia
Лучшее окно 07:00–11:00 UTC. Северный ветер 8 км/ч — чисто оффшорный,
держит волны ровными. Зыбь 0.9м@10с с запада — идеальный размер для pop-up.
После 12:00 ветер поворачивает на SW, становится оншорным.

⭐ ЗАПАСНОЙ: Luz
Те же условия утром, чуть меньше волна из-за укрытия мыса.

❌ ПРОПУСТИТЬ:
• Tonel — SW ветер после полудня, оншорный для западного побережья
• Castelejo — зыбь 1.1м, приемлемо, но далеко (40 мин) и ветер хуже

📅 Завтра: чуть хуже — волна вырастет до 1.2м, ветер усилится до 25 км/ч

📊 Данные: ветер N 8 км/ч, зыбь 0.9м@10с из W, источник: Stormglass
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
    algarve/
      config.json                   # id, display_name, forecast_points, tide_point
      prompt.txt                    # системный промпт для Claude
testdata/
  stormglass_response.json          # fixture прогноза для тестового режима
  stormglass_tides.json             # fixture приливов для тестового режима
```

### Как работает

1. Stormglass возвращает почасовые данные по точкам побережья (06:00–18:00 UTC) и приливные экстремумы за два дня
2. Claude получает все часы по порядку, данные о приливах и сам находит лучшее временное окно для сессии
3. Рекомендация на русском отправляется в Telegram-канал (или в stdout при `--mock`)

### Почему используется поле `.sg` из ответа Stormglass

Stormglass возвращает значения каждого параметра сразу из нескольких погодных моделей:

```json
"waveHeight": { "sg": 0.82, "noaa": 0.75, "icon": 0.88 }
```

`sg` — собственная сводная модель Stormglass, которая блендирует несколько источников в одно значение. Мы берём именно её, потому что:
- она **всегда присутствует** — остальные источники могут отсутствовать для конкретной точки или параметра
- Stormglass уже сделал блендинг — выбирать между моделями самостоятельно нет смысла

---

## Добавить новый регион

1. Создай директорию `config/regions/<name>/`
2. Добавь `config.json` с метаданными и точками прогноза:

```json
{
  "id": "lisbon",
  "display_name": "Лиссабон",
  "tide_point": { "lat": 38.7, "lng": -9.4 },
  "forecast_points": [
    { "name": "Побережье Кашкайш", "lat": 38.7, "lng": -9.4 }
  ]
}
```

3. Добавь `prompt.txt` с системным промптом для Claude — описание спотов, правила интерпретации, формат ответа.

Регион подхватится автоматически при следующем запуске — код менять не нужно.

---

## Споты Алгарве

| Спот | Дорога от Лагуша | Побережье |
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
