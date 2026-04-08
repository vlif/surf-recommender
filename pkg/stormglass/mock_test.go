package stormglass

import (
	"testing"
	"time"
)

func TestShiftDatesToToday(t *testing.T) {
	// Фиксируем "сегодня" для детерминированных проверок.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	fixtureDay := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	fixtureDayPlusOne := fixtureDay.Add(24 * time.Hour)

	makeTime := func(base time.Time, hour int) string {
		return base.Add(time.Duration(hour) * time.Hour).Format(time.RFC3339)
	}

	t.Run("пустой слайс возвращается без изменений", func(t *testing.T) {
		result := shiftDatesToToday([]HourData{})
		if len(result) != 0 {
			t.Errorf("ожидали пустой слайс, получили %d элементов", len(result))
		}
	})

	t.Run("первый день fixture становится сегодня", func(t *testing.T) {
		hours := []HourData{
			{Time: makeTime(fixtureDay, 6)},
			{Time: makeTime(fixtureDay, 7)},
		}

		result := shiftDatesToToday(hours)

		for _, h := range result {
			got, err := time.Parse(time.RFC3339, h.Time)
			if err != nil {
				t.Fatalf("не удалось распарсить время %q: %v", h.Time, err)
			}
			gotDay := got.UTC().Truncate(24 * time.Hour)
			if !gotDay.Equal(today) {
				t.Errorf("ожидали дату %v, получили %v", today, gotDay)
			}
		}
	})

	t.Run("второй день fixture становится завтра", func(t *testing.T) {
		hours := []HourData{
			{Time: makeTime(fixtureDay, 6)},
			{Time: makeTime(fixtureDayPlusOne, 6)},
		}

		result := shiftDatesToToday(hours)

		gotDay0 := mustParseDay(t, result[0].Time)
		gotDay1 := mustParseDay(t, result[1].Time)

		if !gotDay0.Equal(today) {
			t.Errorf("result[0]: ожидали %v, получили %v", today, gotDay0)
		}
		if !gotDay1.Equal(tomorrow) {
			t.Errorf("result[1]: ожидали %v, получили %v", tomorrow, gotDay1)
		}
	})

	t.Run("время внутри дня сохраняется", func(t *testing.T) {
		hours := []HourData{
			{Time: makeTime(fixtureDay, 8)},
			{Time: makeTime(fixtureDay, 13)},
		}

		result := shiftDatesToToday(hours)

		if got := mustParseHour(t, result[0].Time); got != 8 {
			t.Errorf("result[0]: ожидали час 8, получили %d", got)
		}
		if got := mustParseHour(t, result[1].Time); got != 13 {
			t.Errorf("result[1]: ожидали час 13, получили %d", got)
		}
	})

	t.Run("остальные поля HourData не затрагиваются", func(t *testing.T) {
		hours := []HourData{
			{
				Time:       makeTime(fixtureDay, 6),
				WaveHeight: SGValue{SG: 1.5},
				WindSpeed:  SGValue{SG: 3.2},
			},
		}

		result := shiftDatesToToday(hours)

		if result[0].WaveHeight.SG != 1.5 {
			t.Errorf("WaveHeight изменился: ожидали 1.5, получили %v", result[0].WaveHeight.SG)
		}
		if result[0].WindSpeed.SG != 3.2 {
			t.Errorf("WindSpeed изменился: ожидали 3.2, получили %v", result[0].WindSpeed.SG)
		}
	})

	t.Run("невалидное время пропускается без паники", func(t *testing.T) {
		hours := []HourData{
			{Time: makeTime(fixtureDay, 6)},
			{Time: "not-a-date"},
		}

		result := shiftDatesToToday(hours)

		if len(result) != 2 {
			t.Fatalf("ожидали 2 элемента, получили %d", len(result))
		}
		// Валидный элемент сдвинулся.
		gotDay := mustParseDay(t, result[0].Time)
		if !gotDay.Equal(today) {
			t.Errorf("result[0]: ожидали %v, получили %v", today, gotDay)
		}
		// Невалидный остался как был.
		if result[1].Time != "not-a-date" {
			t.Errorf("result[1]: ожидали оригинальное значение, получили %q", result[1].Time)
		}
	})
}

func mustParseDay(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("не удалось распарсить время %q: %v", s, err)
	}
	return parsed.UTC().Truncate(24 * time.Hour)
}

func mustParseHour(t *testing.T, s string) int {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("не удалось распарсить время %q: %v", s, err)
	}
	return parsed.UTC().Hour()
}
