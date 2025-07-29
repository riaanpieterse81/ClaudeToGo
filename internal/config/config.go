package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// Load loads configuration from a JSON file
func Load(path string) (*types.ConfigFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config types.ConfigFile
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Save saves the configuration to a JSON file
func Save(config types.ConfigFile, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// Apply applies configuration file settings to the runtime config
func Apply(configFile *types.ConfigFile, config *types.Config) error {
	// Parse poll interval from string
	if configFile.PollInterval != "" {
		duration, err := time.ParseDuration(configFile.PollInterval)
		if err != nil {
			return err
		}
		config.PollInterval = duration
	}

	// Apply other settings (command line flags will override these later)
	config.LogFile = configFile.LogFile
	config.Verbose = configFile.Verbose

	return nil
}