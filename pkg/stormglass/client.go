package stormglass

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL     = "https://api.stormglass.io/v2/weather/point"
	tideURL     = "https://api.stormglass.io/v2/tide/extremes/point"
	apiParams   = "waveHeight,wavePeriod,waveDirection,windSpeed,windDirection,swellHeight,swellPeriod,swellDirection,secondarySwellHeight,secondarySwellDirection,secondarySwellPeriod,waterTemperature,gust"
)

type SGValue struct {
	SG float64 `json:"sg"`
}

type HourData struct {
	Time                    string  `json:"time"`
	WaveHeight              SGValue `json:"waveHeight"`
	WavePeriod              SGValue `json:"wavePeriod"`
	WaveDirection           SGValue `json:"waveDirection"`
	SwellHeight             SGValue `json:"swellHeight"`
	SwellPeriod             SGValue `json:"swellPeriod"`
	SwellDirection          SGValue `json:"swellDirection"`
	SecondarySwellHeight    SGValue `json:"secondarySwellHeight"`
	SecondarySwellPeriod    SGValue `json:"secondarySwellPeriod"`
	SecondarySwellDirection SGValue `json:"secondarySwellDirection"`
	WindSpeed               SGValue `json:"windSpeed"`
	WindDirection           SGValue `json:"windDirection"`
	WaterTemperature        SGValue `json:"waterTemperature"`
	Gust                    SGValue `json:"gust"`
}

type Response struct {
	Hours []HourData `json:"hours"`
}

// TideExtreme — один экстремум прилива (максимум или минимум).
type TideExtreme struct {
	Time   string  `json:"time"`
	Height float64 `json:"height"`
	Type   string  `json:"type"` // "high" или "low"
}

type tideResponse struct {
	Data []TideExtreme `json:"data"`
}

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Fetch(lat, lng float64, start, end time.Time) ([]HourData, error) {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("lat", fmt.Sprintf("%.3f", lat))
	q.Set("lng", fmt.Sprintf("%.3f", lng))
	q.Set("params", apiParams)
	q.Set("start", start.UTC().Format(time.RFC3339))
	q.Set("end", end.UTC().Format(time.RFC3339))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stormglass request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stormglass API %d: %s", resp.StatusCode, body)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode stormglass response: %w", err)
	}
	return result.Hours, nil
}

// FetchTides возвращает приливные экстремумы (максимумы и минимумы) за период [start, end].
func (c *Client) FetchTides(lat, lng float64, start, end time.Time) ([]TideExtreme, error) {
	u, _ := url.Parse(tideURL)
	q := u.Query()
	q.Set("lat", fmt.Sprintf("%.3f", lat))
	q.Set("lng", fmt.Sprintf("%.3f", lng))
	q.Set("start", start.UTC().Format(time.RFC3339))
	q.Set("end", end.UTC().Format(time.RFC3339))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build tide request: %w", err)
	}
	req.Header.Set("Authorization", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stormglass tide request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stormglass tide API %d: %s", resp.StatusCode, body)
	}

	var result tideResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode tide response: %w", err)
	}
	return result.Data, nil
}

// FilterTideDay возвращает приливные экстремумы за указанную дату (UTC).
func FilterTideDay(tides []TideExtreme, date time.Time) []TideExtreme {
	y, m, d := date.UTC().Date()
	var result []TideExtreme
	for _, t := range tides {
		pt, err := time.Parse(time.RFC3339, t.Time)
		if err != nil {
			continue
		}
		py, pm, pd := pt.UTC().Date()
		if py == y && pm == m && pd == d {
			result = append(result, t)
		}
	}
	return result
}

// FilterDay возвращает часы за указанную дату в диапазоне [startHour, endHour] UTC.
func FilterDay(hours []HourData, date time.Time, startHour, endHour int) []HourData {
	y, m, d := date.UTC().Date()
	var result []HourData
	for _, h := range hours {
		t, err := time.Parse(time.RFC3339, h.Time)
		if err != nil {
			continue
		}
		t = t.UTC()
		ty, tm, td := t.Date()
		if ty == y && tm == m && td == d && t.Hour() >= startHour && t.Hour() <= endHour {
			result = append(result, h)
		}
	}
	return result
}

// DegreesToCompass конвертирует градусы в обозначение стороны света.
func DegreesToCompass(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int(math.Round(deg/22.5)) % 16
	return dirs[idx]
}
