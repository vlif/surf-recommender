package regions

// ForecastPoint — точка для запроса прогноза в Stormglass API.
// Обычно регион содержит 2-3 точки, представляющие разные участки побережья.
type ForecastPoint struct {
	// Name используется в тексте промпта для Claude — называй понятно.
	// Например: "Южное побережье (Meia Praia, Luz, Burgau)"
	Name string
	Lat  float64
	Lng  float64
}

// Region описывает регион для рекомендаций сёрфинга.
// Чтобы добавить новый регион — создай пакет internal/regions/<name>/region.go
// и определи там переменную Region этого типа.
type Region struct {
	// ID — машинный идентификатор, используется в флаге --region.
	// Пример: "algarve", "lisbon"
	ID string

	// DisplayName — человекочитаемое название для логов и заголовков.
	DisplayName string

	// SystemPrompt — полный системный промпт для Claude:
	// профиль пользователя, база спотов, правила интерпретации, формат ответа.
	// Уникален для каждого региона.
	SystemPrompt string

	// ForecastPoints — список точек для запроса в Stormglass.
	// Сервис запросит прогноз для каждой точки и передаст данные Claude.
	ForecastPoints []ForecastPoint
}
