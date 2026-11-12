package config

import (
	"fmt"
	"math"
)

type TrajectoryConfig struct {
	LookaheadHours            int     `yaml:"lookahead_hours"`
	SlopeThresholdUGM3PerHour float64 `yaml:"slope_threshold_ug_m3_per_hour"`
}

type OutlookConfig struct {
	Version    int              `yaml:"version"`
	Trajectory TrajectoryConfig `yaml:"trajectory"`
}

func LoadOutlookConfig(path string) (OutlookConfig, error) {
	var config OutlookConfig
	if err := decodeStrictYAML(path, &config); err != nil {
		return OutlookConfig{}, err
	}
	if err := config.Validate(); err != nil {
		return OutlookConfig{}, fmt.Errorf("validate outlook config %q: %w", path, err)
	}
	return config, nil
}

func (c OutlookConfig) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported outlook config version %d", c.Version)
	}
	if c.Trajectory.LookaheadHours != 6 {
		return fmt.Errorf("trajectory lookahead must be exactly six clock hours")
	}
	threshold := c.Trajectory.SlopeThresholdUGM3PerHour
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold != 2 {
		return fmt.Errorf("trajectory slope threshold must be exactly 2 ug/m3 per hour")
	}
	return nil
}
