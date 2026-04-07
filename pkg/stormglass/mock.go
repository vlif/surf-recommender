package stormglass

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// MockClient читает данные прогноза из локального JSON-файла.
// Формат файла совпадает с ответом Stormglass API (поле "hours").
// Все даты в fixture автоматически сдвигаются на текущий день,
// поэтому фильтрация в сервисе всегда находит данные.
type MockClient struct {
	filePath string
}

func NewMockClient(filePath string) *MockClient {
	return &MockClient{filePath: filePath}
}

// Fetch игнорирует lat/lng и возвращает данные из файла с пересчитанными датами.
func (m *MockClient) Fetch(_, _ float64, _, _ time.Time) ([]HourData, error) {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("mock: read fixture %s: %w", m.filePath, err)
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("mock: decode fixture: %w", err)
	}

	return shiftDatesToToday(resp.Hours), nil
}

// shiftDatesToToday сдвигает все временные метки fixture так,
// чтобы первый день стал сегодня, второй — завтра и т.д.
func shiftDatesToToday(hours []HourData) []HourData {
	if len(hours) == 0 {
		return hours
	}

	firstTime, err := time.Parse(time.RFC3339, hours[0].Time)
	if err != nil {
		return hours
	}

	firstDay := firstTime.UTC().Truncate(24 * time.Hour)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	offset := today.Sub(firstDay)

	result := make([]HourData, len(hours))
	for i, h := range hours {
		t, err := time.Parse(time.RFC3339, h.Time)
		if err != nil {
			result[i] = h
			continue
		}
		h.Time = t.Add(offset).Format(time.RFC3339)
		result[i] = h
	}
	return result
}
