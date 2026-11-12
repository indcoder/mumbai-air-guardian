package config

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOutlookConfigUsesApprovedTrajectoryWindowAndThreshold(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	config, err := LoadOutlookConfig(filepath.Join(root, "config", "outlook.yaml"))
	if err != nil {
		t.Fatalf("load outlook config: %v", err)
	}
	if config.Trajectory.LookaheadHours != 6 || config.Trajectory.SlopeThresholdUGM3PerHour != 2 {
		t.Fatalf("unexpected trajectory configuration: %+v", config.Trajectory)
	}
}

func TestOutlookConfigRejectsChangedOrInvalidTrajectory(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*OutlookConfig)
	}{
		{"extended lookahead", func(c *OutlookConfig) { c.Trajectory.LookaheadHours = 7 }},
		{"negative threshold", func(c *OutlookConfig) { c.Trajectory.SlopeThresholdUGM3PerHour = -2 }},
		{"non-finite threshold", func(c *OutlookConfig) { c.Trajectory.SlopeThresholdUGM3PerHour = math.Inf(1) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := OutlookConfig{Version: 1, Trajectory: TrajectoryConfig{
				LookaheadHours: 6, SlopeThresholdUGM3PerHour: 2,
			}}
			test.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("invalid trajectory configuration was accepted")
			}
		})
	}
}
