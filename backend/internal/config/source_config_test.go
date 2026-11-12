package config

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadSourceConfigMatchesApprovedImmutableInputs(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	config, err := LoadSourceConfig(filepath.Join(root, "config", "sources.yaml"))
	if err != nil {
		t.Fatalf("load approved source config: %v", err)
	}
	if config.Forecasts.GenerationTime != "unavailable" {
		t.Fatalf("generation time must remain unavailable, got %q", config.Forecasts.GenerationTime)
	}
	if config.OptionalEnvironmentalContext.Enabled {
		t.Fatal("unvalidated environmental context must remain disabled")
	}
}

func TestSourceConfigRejectsAlternativeDataSources(t *testing.T) {
	config, err := LoadSourceConfig(filepath.Join("..", "..", "..", "config", "sources.yaml"))
	if err != nil {
		t.Fatalf("load source config: %v", err)
	}
	config.Observations.Path = filepath.Join("zindi_artifacts", "test_train_overlap.csv")
	if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "observations path") {
		t.Fatalf("alternate observation source was not rejected: %v", err)
	}
}
