package stormglass

import "time"

// Fetcher — интерфейс для получения данных прогноза.
// Реализации:
//   - Client      — реальный Stormglass API
//   - MockClient  — из JSON-файла, для тестов (не тратит квоту)
type Fetcher interface {
	Fetch(lat, lng float64, start, end time.Time) ([]HourData, error)
}

// TideFetcher — интерфейс для получения данных о приливах.
// Реализации:
//   - Client          — реальный Stormglass API (/tide/extremes/point)
//   - MockTideClient  — из JSON-файла, для тестов
type TideFetcher interface {
	FetchTides(lat, lng float64, start, end time.Time) ([]TideExtreme, error)
}
