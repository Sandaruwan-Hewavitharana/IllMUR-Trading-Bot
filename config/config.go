package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config holds all runtime configuration.
type Config struct {
	// AI API Keys
	GeminiKeys  []string `json:"gemini_keys"`
	MistralKeys []string `json:"mistral_keys"`

	// Chrome / Chromedp
	ChromeUserDataDir   string `json:"chrome_user_data_dir"`
	ChromeProfileName   string `json:"chrome_profile_name"`
	TradingViewURL      string `json:"tradingview_url"`
	ChartRenderWaitSecs int    `json:"chart_render_wait_secs"`
	LoopIntervalMinutes int    `json:"loop_interval_minutes"`

	// MT5 Bridge Endpoint
	MT5Endpoint string `json:"mt5_endpoint"`
}

var (
	settingsMutex sync.RWMutex
	settingsPath  string
)

func init() {
	exePath, err := os.Executable()
	if err != nil {
		settingsPath = "settings.json"
	} else {
		settingsPath = filepath.Join(filepath.Dir(exePath), "settings.json")
	}
}

// Load reads configuration from settings.json.
// If it doesn't exist, it creates a default configuration.
func Load() (*Config, error) {
	settingsMutex.RLock()
	data, err := os.ReadFile(settingsPath)
	settingsMutex.RUnlock()

	if err != nil {
		if os.IsNotExist(err) {
			return createDefaultConfig()
		}
		// Also try current directory if running via wails dev
		data, err = os.ReadFile("settings.json")
		if err != nil {
			if os.IsNotExist(err) {
				return createDefaultConfig()
			}
			return nil, fmt.Errorf("failed to read settings.json: %w", err)
		}
		settingsPath = "settings.json" // Update path to current dir for dev mode
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse settings.json: %w", err)
	}

	// Auto-migrate legacy hardcoded paths to the new portable dynamic path
	if cfg.ChromeUserDataDir == `D:\IllMUR\GoldBot\user_data` || cfg.ChromeUserDataDir == "" {
		cfg.ChromeUserDataDir = filepath.Join(filepath.Dir(settingsPath), "isolated_bot_data")
		Save(cfg) // Silently update the settings file
	}

	return cfg, nil
}

// Save writes the configuration to settings.json
func Save(cfg *Config) error {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}
	return nil
}

func createDefaultConfig() (*Config, error) {
	// Resolve the bot data directory relative to the executable
	dataDir := filepath.Join(filepath.Dir(settingsPath), "isolated_bot_data")

	cfg := &Config{
		GeminiKeys:          []string{},
		MistralKeys:         []string{},
		ChromeUserDataDir:   dataDir,
		ChromeProfileName:   "Default",
		TradingViewURL:      "https://www.tradingview.com/chart/5pUKnxDY/?symbol=OANDA%3AXAUUSD",
		ChartRenderWaitSecs: 30,
		LoopIntervalMinutes: 5,
		MT5Endpoint:         "http://127.0.0.1:8080/order",
	}
	err := Save(cfg)
	return cfg, err
}
