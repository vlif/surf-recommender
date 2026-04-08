package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"surf-recommender/internal/regions"
	"strings"
)

// LoadRegion загружает регион из директории regionDir.
// Ожидает два файла:
//   - config.json — метаданные и точки прогноза
//   - prompt.txt  — системный промпт для Claude
func LoadRegion(regionDir string) (regions.Region, error) {
	cfgData, err := os.ReadFile(filepath.Join(regionDir, "config.json"))
	if err != nil {
		return regions.Region{}, fmt.Errorf("читаю config.json: %w", err)
	}

	var region regions.Region
	if err := json.Unmarshal(cfgData, &region); err != nil {
		return regions.Region{}, fmt.Errorf("парсю config.json: %w", err)
	}

	promptData, err := os.ReadFile(filepath.Join(regionDir, "prompt.txt"))
	if err != nil {
		return regions.Region{}, fmt.Errorf("читаю prompt.txt: %w", err)
	}
	region.SystemPrompt = strings.TrimSpace(string(promptData))

	if region.ID == "" {
		return regions.Region{}, fmt.Errorf("поле id не задано в config.json")
	}
	if len(region.ForecastPoints) == 0 {
		return regions.Region{}, fmt.Errorf("forecast_points пустой в config.json")
	}
	if region.SystemPrompt == "" {
		return regions.Region{}, fmt.Errorf("prompt.txt пустой")
	}

	return region, nil
}

// LoadAllRegions сканирует baseDir и загружает все регионы из поддиректорий.
// Возвращает map[regionID]Region.
func LoadAllRegions(baseDir string) (map[string]regions.Region, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("читаю директорию регионов %s: %w", baseDir, err)
	}

	registry := make(map[string]regions.Region)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		regionDir := filepath.Join(baseDir, entry.Name())
		region, err := LoadRegion(regionDir)
		if err != nil {
			return nil, fmt.Errorf("загружаю регион %s: %w", entry.Name(), err)
		}
		registry[region.ID] = region
	}

	if len(registry) == 0 {
		return nil, fmt.Errorf("нет ни одного региона в %s", baseDir)
	}

	return registry, nil
}
