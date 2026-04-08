package regions

// ForecastPoint — точка для запроса прогноза в Stormglass API.
type ForecastPoint struct {
	// Name используется в тексте промпта для Claude — называй понятно.
	// Например: "Южное побережье (Meia Praia, Luz, Burgau)"
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// TidePoint — точка для запроса приливного прогноза.
// Достаточно одной точки на весь регион — приливы одинаковы на десятки км.
type TidePoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Region описывает регион для рекомендаций сёрфинга.
// Данные хранятся в config/regions/<id>/:
//   - config.json — id, display_name, forecast_points, tide_point
//   - prompt.txt  — системный промпт для Claude
type Region struct {
	ID             string          `json:"id"`
	DisplayName    string          `json:"display_name"`
	SystemPrompt   string          `json:"-"` // загружается из prompt.txt
	ForecastPoints []ForecastPoint `json:"forecast_points"`
	TidePoint      *TidePoint      `json:"tide_point"` // nil — приливы не запрашиваются
}
