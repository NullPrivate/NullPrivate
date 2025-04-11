// Package ruleset provides functionality for managing rule sets.
package ruleset

import (
	"fmt"
	"os"
)

// Config represents ruleset configuration settings.
type Config struct {
	BaseDir string `yaml:"base_dir"`
}

type Ruleset struct {
	// BaseDir is the base directory for storing rulesets.
	// If not specified, a default directory will be used.
	BaseDir string `yaml:"base_dir"`
	conf    *Config
}

// New creates a new Ruleset instance with the specified configuration.
func New(conf *Config) (*Ruleset, error) {
	if conf == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if conf.BaseDir == "" {
		return nil, fmt.Errorf("base directory cannot be empty")
	}
	// Additional validation can be added here if needed.
	// Create the base directory if it doesn't exist
	if _, err := os.Stat(conf.BaseDir); os.IsNotExist(err) {
		if dirErr := os.MkdirAll(conf.BaseDir, 0o755); dirErr != nil {
			return nil, fmt.Errorf("failed to create base directory: %v", err)
		}
	}
	// Initialize the Ruleset instance
	// with the provided configuration.
	return &Ruleset{
		BaseDir: conf.BaseDir,
		conf:    conf,
	}, nil
}
