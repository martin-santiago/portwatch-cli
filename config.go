package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Config struct {
	FilterPorts            []int `json:"filter_ports"`
	FilterEnabled          bool  `json:"filter_enabled"`
	RefreshIntervalSeconds int   `json:"refresh_interval_seconds"`
}

var (
	appConfig  Config
	configMu   sync.RWMutex
	configPath string
)

func initConfig() {
	home, _ := os.UserHomeDir()
	configPath = filepath.Join(home, ".portwatch.json")

	configMu.Lock()
	defer configMu.Unlock()

	appConfig = Config{
		FilterPorts:            []int{3001, 3002, 3003, 3005, 7000, 8000},
		FilterEnabled:          false,
		RefreshIntervalSeconds: 3,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		saveConfigLocked()
		return
	}

	if err := json.Unmarshal(data, &appConfig); err != nil {
		saveConfigLocked()
		return
	}
}

func getConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()

	c := appConfig
	ports := make([]int, len(appConfig.FilterPorts))
	copy(ports, appConfig.FilterPorts)
	c.FilterPorts = ports
	return c
}

func setFilterEnabled(enabled bool) {
	configMu.Lock()
	defer configMu.Unlock()
	appConfig.FilterEnabled = enabled
	saveConfigLocked()
}

func toggleFilter() bool {
	configMu.Lock()
	defer configMu.Unlock()
	appConfig.FilterEnabled = !appConfig.FilterEnabled
	saveConfigLocked()
	return appConfig.FilterEnabled
}

func addFilterPort(port int) {
	configMu.Lock()
	defer configMu.Unlock()
	for _, p := range appConfig.FilterPorts {
		if p == port {
			return
		}
	}
	appConfig.FilterPorts = append(appConfig.FilterPorts, port)
	sort.Ints(appConfig.FilterPorts)
	saveConfigLocked()
}

func removeFilterPort(port int) {
	configMu.Lock()
	defer configMu.Unlock()
	for i, p := range appConfig.FilterPorts {
		if p == port {
			appConfig.FilterPorts = append(appConfig.FilterPorts[:i], appConfig.FilterPorts[i+1:]...)
			saveConfigLocked()
			return
		}
	}
}

func saveConfigLocked() {
	data, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(configPath, data, 0644)
}
