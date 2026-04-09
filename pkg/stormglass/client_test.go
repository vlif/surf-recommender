package stormglass

import (
	"testing"
	"time"
)

// --- DegreesToCompass ---

func TestDegreesToCompass(t *testing.T) {
	cases := []struct {
		deg  float64
		want string
	}{
		// Основные стороны света
		{0, "N"},
		{90, "E"},
		{180, "S"},
		{270, "W"},
		// Промежуточные
		{45, "NE"},
		{135, "SE"},
		{225, "SW"},
		{315, "NW"},
		// Точно на границе секторов (22.5 → NNE)
		{22.5, "NNE"},
		{67.5, "ENE"},
		// 360° = 0° = N
		{360, "N"},
		// Округление: 11° ближе к N (0°), 12° ближе к NNE (22.5°)
		{11, "N"},
		{12, "NNE"},
		// Типичные серф-условия
		{272, "W"},   // свелл с запада
		{8, "N"},     // северный ветер
		{200, "SSW"}, // юго-юго-запад
	}

	for _, c := range cases {
		got := DegreesToCompass(c.deg)
		if got != c.want {
			t.Errorf("DegreesToCompass(%.1f) = %q, хотели %q", c.deg, got, c.want)
		}
	}
}

// --- FilterDay ---

func TestFilterDay(t *testing.T) {
	day := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)
	nextDay := day.Add(24 * time.Hour)

	makeHour := func(t time.Time) HourData {
		return HourData{Time: t.Format(time.RFC3339)}
	}

	t.Run("пустой слайс возвращает nil", func(t *testing.T) {
		result := FilterDay(nil, day, 6, 18)
		if len(result) != 0 {
			t.Errorf("ожидали пустой результат, получили %d", len(result))
		}
	})

	t.Run("возвращает часы только за нужную дату", func(t *testing.T) {
		hours := []HourData{
			makeHour(day.Add(8 * time.Hour)),
			makeHour(nextDay.Add(8 * time.Hour)), // завтра — не должен попасть
		}
		result := FilterDay(hours, day, 6, 18)
		if len(result) != 1 {
			t.Fatalf("ожидали 1 запись, получили %d", len(result))
		}
	})

	t.Run("граничные часы включаются [startHour, endHour]", func(t *testing.T) {
		hours := []HourData{
			makeHour(day.Add(5 * time.Hour)),  // до окна — не должен попасть
			makeHour(day.Add(6 * time.Hour)),  // ровно startHour — должен
			makeHour(day.Add(18 * time.Hour)), // ровно endHour — должен
			makeHour(day.Add(19 * time.Hour)), // после окна — не должен
		}
		result := FilterDay(hours, day, 6, 18)
		if len(result) != 2 {
			t.Errorf("ожидали 2 записи (06:00 и 18:00), получили %d", len(result))
		}
	})

	t.Run("невалидное время пропускается", func(t *testing.T) {
		hours := []HourData{
			makeHour(day.Add(8 * time.Hour)),
			{Time: "not-a-date"},
		}
		result := FilterDay(hours, day, 6, 18)
		if len(result) != 1 {
			t.Errorf("ожидали 1 запись, получили %d", len(result))
		}
	})

	t.Run("время с таймзоной нормализуется в UTC", func(t *testing.T) {
		// 09:00+01:00 = 08:00 UTC — должен попасть в окно 06–18
		hours := []HourData{
			{Time: "2026-04-09T09:00:00+01:00"},
		}
		result := FilterDay(hours, day, 6, 18)
		if len(result) != 1 {
			t.Errorf("ожидали 1 запись после нормализации UTC, получили %d", len(result))
		}
	})
}

// --- FilterTideDay ---

func TestFilterTideDay(t *testing.T) {
	day := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)
	nextDay := day.Add(24 * time.Hour)

	makeTide := func(t time.Time, tideType string, height float64) TideExtreme {
		return TideExtreme{
			Time:   t.Format(time.RFC3339),
			Type:   tideType,
			Height: height,
		}
	}

	t.Run("возвращает только приливы за нужный день", func(t *testing.T) {
		tides := []TideExtreme{
			makeTide(day.Add(6*time.Hour), "high", 2.0),
			makeTide(day.Add(12*time.Hour), "low", 0.4),
			makeTide(nextDay.Add(6*time.Hour), "high", 1.9), // завтра — не должен попасть
		}
		result := FilterTideDay(tides, day)
		if len(result) != 2 {
			t.Fatalf("ожидали 2 прилива за день, получили %d", len(result))
		}
	})

	t.Run("пустой слайс возвращает nil", func(t *testing.T) {
		result := FilterTideDay(nil, day)
		if len(result) != 0 {
			t.Errorf("ожидали пустой результат, получили %d", len(result))
		}
	})

	t.Run("невалидное время пропускается", func(t *testing.T) {
		tides := []TideExtreme{
			makeTide(day.Add(10*time.Hour), "low", 0.3),
			{Time: "bad-date", Type: "high", Height: 2.0},
		}
		result := FilterTideDay(tides, day)
		if len(result) != 1 {
			t.Errorf("ожидали 1 запись, получили %d", len(result))
		}
	})

	t.Run("сохраняет тип и высоту прилива", func(t *testing.T) {
		tides := []TideExtreme{
			makeTide(day.Add(4*time.Hour), "high", 2.1),
			makeTide(day.Add(10*time.Hour), "low", 0.4),
		}
		result := FilterTideDay(tides, day)

		if result[0].Type != "high" || result[0].Height != 2.1 {
			t.Errorf("первый прилив: ожидали high/2.1, получили %s/%.1f", result[0].Type, result[0].Height)
		}
		if result[1].Type != "low" || result[1].Height != 0.4 {
			t.Errorf("второй прилив: ожидали low/0.4, получили %s/%.1f", result[1].Type, result[1].Height)
		}
	})
}
