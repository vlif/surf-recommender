package spotrecommender

import (
	"fmt"
	"strings"
	"surf-recommender/internal/regions"
	"surf-recommender/pkg/anthropic"
	"surf-recommender/pkg/stormglass"
	"time"
)

// Временной диапазон данных, которые передаём Claude.
// 06:00–18:00 UTC = 07:00–19:00 по Лиссабону — весь световой сёрф-день.
const (
	dayStartHour = 6
	dayEndHour   = 18
)

// Service координирует получение прогноза и генерацию рекомендации для заданного региона.
type Service struct {
	sg        stormglass.Fetcher
	tides     stormglass.TideFetcher // nil — приливы не запрашиваются
	anthropic *anthropic.Client
	region    regions.Region
}

func NewService(sg stormglass.Fetcher, tides stormglass.TideFetcher, anthropic *anthropic.Client, region regions.Region) *Service {
	return &Service{sg: sg, tides: tides, anthropic: anthropic, region: region}
}

// pointHours — почасовые данные одной точки прогноза за один день.
type pointHours struct {
	point    regions.ForecastPoint
	today    []stormglass.HourData
	tomorrow []stormglass.HourData
}

// Recommend получает прогноз для всех точек региона и возвращает рекомендацию от Claude.
func (s *Service) Recommend(spot string, date time.Time) (string, error) {
	today := date.UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	// Один запрос к Stormglass покрывает и сегодня, и завтра.
	start := today.Add(time.Duration(dayStartHour) * time.Hour)
	end := tomorrow.Add(time.Duration(dayEndHour) * time.Hour)

	data, err := s.fetchAllPoints(start, end, today, tomorrow)
	if err != nil {
		return "", err
	}

	if len(data[0].today) == 0 {
		return "", fmt.Errorf("нет данных Stormglass для даты %s", today.Format("2006-01-02"))
	}

	var todayTides, tomorrowTides []stormglass.TideExtreme
	if s.tides != nil && s.region.TidePoint != nil {
		tp := s.region.TidePoint
		// Запрашиваем с запасом: от полуночи сегодня до полуночи послезавтра.
		allTides, err := s.tides.FetchTides(tp.Lat, tp.Lng, today, tomorrow.Add(24*time.Hour))
		if err != nil {
			return "", fmt.Errorf("fetch tides: %w", err)
		}
		todayTides = stormglass.FilterTideDay(allTides, today)
		tomorrowTides = stormglass.FilterTideDay(allTides, tomorrow)
	}

	userMsg := buildUserMessage(spot, today, data, todayTides, tomorrowTides)
	return s.anthropic.Send(s.region.SystemPrompt, userMsg)
}

// fetchAllPoints делает один запрос на точку и разбивает ответ по дням.
func (s *Service) fetchAllPoints(start, end, today, tomorrow time.Time) ([]pointHours, error) {
	result := make([]pointHours, len(s.region.ForecastPoints))
	for i, point := range s.region.ForecastPoints {
		result[i].point = point
		hours, err := s.sg.Fetch(point.Lat, point.Lng, start, end)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", point.Name, err)
		}
		result[i].today = stormglass.FilterDay(hours, today, dayStartHour, dayEndHour)
		result[i].tomorrow = stormglass.FilterDay(hours, tomorrow, dayStartHour, dayEndHour)
	}
	return result, nil
}

func buildUserMessage(spot string, date time.Time, data []pointHours, todayTides, tomorrowTides []stormglass.TideExtreme) string {
	spotLine := "Порекомендуй лучший спот для сёрфинга сегодня."
	if spot != "" {
		spotLine = fmt.Sprintf("Проанализируй условия конкретно для спота %s и сравни с альтернативами.", spot)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Дата: %s\n", date.Format("Monday, 2 January 2006"))
	fmt.Fprintf(&sb, "Данные: %02d:00–%02d:00 UTC (%02d:00–%02d:00 по Лиссабону), почасово\n\n",
		dayStartHour, dayEndHour, dayStartHour+1, dayEndHour+1)
	fmt.Fprintf(&sb, "%s\n", spotLine)

	if len(todayTides) > 0 {
		fmt.Fprintf(&sb, "\n=== ПРИЛИВЫ СЕГОДНЯ (UTC) ===\n")
		fmt.Fprintf(&sb, "%s\n", formatTides(todayTides))
	}
	if len(tomorrowTides) > 0 {
		fmt.Fprintf(&sb, "\n=== ПРИЛИВЫ ЗАВТРА (UTC) ===\n")
		fmt.Fprintf(&sb, "%s\n", formatTides(tomorrowTides))
	}

	fmt.Fprintf(&sb, "\n=== ПРОГНОЗ СЕГОДНЯ ===\n")
	for _, p := range data {
		if len(p.today) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "\n%s:\n", p.point.Name)
		fmt.Fprintf(&sb, "%s\n", formatHourly(p.today))
	}

	hasTomorrow := false
	for _, p := range data {
		if len(p.tomorrow) > 0 {
			hasTomorrow = true
			break
		}
	}
	if hasTomorrow {
		fmt.Fprintf(&sb, "\n=== ПРОГНОЗ ЗАВТРА ===\n")
		for _, p := range data {
			if len(p.tomorrow) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "\n%s:\n", p.point.Name)
			fmt.Fprintf(&sb, "%s\n", formatHourly(p.tomorrow))
		}
	}

	return sb.String()
}

func formatTides(tides []stormglass.TideExtreme) string {
	var sb strings.Builder
	for _, t := range tides {
		pt, err := time.Parse(time.RFC3339, t.Time)
		if err != nil {
			continue
		}
		tideType := "максимум"
		if t.Type == "low" {
			tideType = "минимум"
		}
		fmt.Fprintf(&sb, "  %s — %s (%.1fм)\n", pt.UTC().Format("15:04"), tideType, t.Height)
	}
	return sb.String()
}

func formatHourly(hours []stormglass.HourData) string {
	var sb strings.Builder
	for _, h := range hours {
		t, err := time.Parse(time.RFC3339, h.Time)
		if err != nil {
			continue
		}
		windKmh := h.WindSpeed.SG * 3.6
		gustKmh := h.Gust.SG * 3.6
		fmt.Fprintf(&sb,
			"  %s | волна %.1fм@%.0fс из %-3s | свелл %.1fм@%.0fс из %-3s | ветер %-3s %4.0f км/ч (пор. %3.0f) | доп.свелл %.1fм из %s\n",
			t.UTC().Format("15:04"),
			h.WaveHeight.SG, h.WavePeriod.SG, stormglass.DegreesToCompass(h.WaveDirection.SG),
			h.SwellHeight.SG, h.SwellPeriod.SG, stormglass.DegreesToCompass(h.SwellDirection.SG),
			stormglass.DegreesToCompass(h.WindDirection.SG), windKmh, gustKmh,
			h.SecondarySwellHeight.SG, stormglass.DegreesToCompass(h.SecondarySwellDirection.SG),
		)
	}
	return sb.String()
}
