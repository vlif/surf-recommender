package stormglass

import "time"

// Fetcher — интерфейс для получения данных прогноза.
// Реализации:
//   - Client      — реальный Stormglass API
//   - MockClient  — из JSON-файла, для тестов (не тратит квоту)
type Fetcher interface {
	Fetch(lat, lng float64, start, end time.Time) ([]HourData, error)
}
