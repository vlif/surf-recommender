package spotrecommender

import (
	"strings"
	"testing"

	"surf-recommender/pkg/stormglass"
)

func TestFormatHourly(t *testing.T) {
	t.Run("пустой слайс возвращает пустую строку", func(t *testing.T) {
		result := formatHourly([]stormglass.HourData{})
		if result != "" {
			t.Errorf("ожидали пустую строку, получили %q", result)
		}
	})

	t.Run("невалидное время пропускается", func(t *testing.T) {
		hours := []stormglass.HourData{
			{Time: "not-a-date"},
		}
		result := formatHourly(hours)
		if result != "" {
			t.Errorf("ожидали пустую строку, получили %q", result)
		}
	})

	t.Run("одна запись форматируется корректно", func(t *testing.T) {
		hours := []stormglass.HourData{
			{
				Time:                    "2026-04-09T08:00:00Z",
				WaveHeight:              stormglass.SGValue{SG: 0.9},
				WavePeriod:              stormglass.SGValue{SG: 10.0},
				WaveDirection:           stormglass.SGValue{SG: 270}, // W
				SwellHeight:             stormglass.SGValue{SG: 0.8},
				SwellPeriod:             stormglass.SGValue{SG: 11.0},
				SwellDirection:          stormglass.SGValue{SG: 270}, // W
				WindDirection:           stormglass.SGValue{SG: 0},   // N
				WindSpeed:               stormglass.SGValue{SG: 3.0}, // 10.8 км/ч
				Gust:                    stormglass.SGValue{SG: 4.0}, // 14.4 км/ч
				SecondarySwellHeight:    stormglass.SGValue{SG: 0.2},
				SecondarySwellDirection: stormglass.SGValue{SG: 180}, // S
			},
		}

		result := formatHourly(hours)
		lines := strings.Split(strings.TrimRight(result, "\n"), "\n")

		if len(lines) != 1 {
			t.Fatalf("ожидали 1 строку, получили %d: %v", len(lines), lines)
		}

		line := lines[0]

		checks := []struct {
			name    string
			contain string
		}{
			{"время UTC", "08:00"},
			{"высота волны", "0.9м"},
			{"период волны", "10с"},
			{"направление волны", "W"},
			{"высота свелла", "0.8м"},
			{"период свелла", "11с"},
			{"ветер N", "N  "},
			{"скорость ветра км/ч", "11 км/ч"},
			{"доп свелл высота", "0.2м"},
			{"доп свелл направление", "S"},
		}

		for _, c := range checks {
			if !strings.Contains(line, c.contain) {
				t.Errorf("[%s] строка не содержит %q:\n%s", c.name, c.contain, line)
			}
		}
	})

	t.Run("скорость ветра конвертируется из м/с в км/ч", func(t *testing.T) {
		hours := []stormglass.HourData{
			{
				Time:      "2026-04-09T08:00:00Z",
				WindSpeed: stormglass.SGValue{SG: 5.0}, // 5 м/с = 18 км/ч
				Gust:      stormglass.SGValue{SG: 7.0}, // 7 м/с = 25 км/ч
			},
		}

		result := formatHourly(hours)

		if !strings.Contains(result, "18 км/ч") {
			t.Errorf("ожидали 18 км/ч (5 м/с × 3.6), строка: %s", result)
		}
		if !strings.Contains(result, " 25") {
			t.Errorf("ожидали порыв 25 км/ч (7 м/с × 3.6), строка: %s", result)
		}
	})

	t.Run("несколько записей — каждая на отдельной строке", func(t *testing.T) {
		hours := []stormglass.HourData{
			{Time: "2026-04-09T06:00:00Z"},
			{Time: "2026-04-09T07:00:00Z"},
			{Time: "2026-04-09T08:00:00Z"},
		}

		result := formatHourly(hours)
		lines := strings.Split(strings.TrimRight(result, "\n"), "\n")

		if len(lines) != 3 {
			t.Errorf("ожидали 3 строки, получили %d", len(lines))
		}

		times := []string{"06:00", "07:00", "08:00"}
		for i, want := range times {
			if !strings.Contains(lines[i], want) {
				t.Errorf("строка %d: ожидали время %s, получили: %s", i, want, lines[i])
			}
		}
	})

	t.Run("время нормализуется в UTC", func(t *testing.T) {
		// +01:00 → UTC = 07:00
		hours := []stormglass.HourData{
			{Time: "2026-04-09T08:00:00+01:00"},
		}

		result := formatHourly(hours)

		if !strings.Contains(result, "07:00") {
			t.Errorf("ожидали 07:00 UTC, строка: %s", result)
		}
	})
}
