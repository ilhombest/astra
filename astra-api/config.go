package main

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

var configMu sync.RWMutex

func jsonConfigPath() string {
	p := *flagConfig
	if strings.HasSuffix(p, ".lua") {
		return p[:len(p)-4] + ".json"
	}
	return p + ".json"
}

func loadConfig() (map[string]any, error) {
	configMu.RLock()
	defer configMu.RUnlock()

	data, err := os.ReadFile(jsonConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{
				"streams":  map[string]any{},
				"settings": map[string]any{},
			}, nil
		}
		return nil, err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func saveConfig(cfg map[string]any) error {
	configMu.Lock()
	defer configMu.Unlock()

	if err := os.MkdirAll(dirOf(jsonConfigPath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonConfigPath(), data, 0644)
}

func dirOf(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "."
	}
	return path[:i]
}
