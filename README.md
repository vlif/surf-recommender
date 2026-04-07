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
cmd/surf-recommender/main.go        # CLI: флаги --region --spot --date --mock
internal/
  regions/
    region.go                       # тип Region и ForecastPoint
    algarve/region.go               # споты, промпт и точки прогноза для Алгарве
  spot-recommender/
    service.go                      # бизнес-логика: fetch → Claude → результат
    commands.go                     # отправка результата в Telegram
pkg/
  stormglass/
    fetcher.go                      # интерфейс Fetcher (реальный и mock)
    client.go                       # реальный Stormglass API клиент
    mock.go                         # mock-клиент из JSON-файла
  anthropic/client.go               # Anthropic Messages API клиент
  telegram/client.go                # Telegram Bot API клиент
testdata/
  stormglass_response.json          # fixture для тестового режима
```

### Как работает

1. Stormglass возвращает почасовые данные по двум точкам побережья (06:00–18:00 UTC)
2. Claude получает все часы по порядку и сам находит лучшее временное окно для сессии
3. Рекомендация на русском отправляется в Telegram-канал

---

## Добавить новый регион

1. Создай `internal/regions/<name>/region.go` с описанием спотов, точками прогноза и промптом
2. Зарегистрируй в `cmd/surf-recommender/main.go`:

```go
var registry = map[string]regions.Region{
    "algarve": algarve.Region,
    "lisbon":  lisbon.Region, // новый регион
}
```

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
