package config

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SourceConfigRelativePath  = "config/sources.yaml"
	RiskBandsRelativePath     = "config/risk-bands.yaml"
	OutlookConfigRelativePath = "config/outlook.yaml"
)

type SourceFile struct {
	Path         string `yaml:"path"`
	SHA256       string `yaml:"sha256"`
	ExpectedRows int    `yaml:"expected_rows"`
}

type ForecastSource struct {
	SourceFile                    `yaml:",inline"`
	SelectedScalingAlreadyApplied float64 `yaml:"selected_scaling_already_applied"`
	GenerationTime                string  `yaml:"generation_time"`
}

type OptionalContextConfig struct {
	Enabled bool   `yaml:"enabled"`
	Status  string `yaml:"status"`
}

type ClimateVerseCatalogue struct {
	DOI           string `yaml:"doi" json:"doi"`
	Version       string `yaml:"version" json:"version"`
	Stage         string `yaml:"stage" json:"stage"`
	DiscoveredVia string `yaml:"discovered_via" json:"discoveredVia"`
	CatalogueRole string `yaml:"catalogue_role" json:"catalogueRole"`
	DataProvider  string `yaml:"data_provider" json:"dataProvider"`
}

type SourceConfig struct {
	Version                      int                   `yaml:"version"`
	Timezone                     string                `yaml:"timezone"`
	Observations                 SourceFile            `yaml:"observations"`
	Forecasts                    ForecastSource        `yaml:"forecasts"`
	ForecastIntervals            SourceFile            `yaml:"forecast_intervals"`
	Stations                     SourceFile            `yaml:"stations"`
	OptionalEnvironmentalContext OptionalContextConfig `yaml:"optional_environmental_context"`
	ClimateVerseCatalogue        ClimateVerseCatalogue `yaml:"climateverse_catalogue"`
}

type RiskBand struct {
	Name           string   `yaml:"name" json:"name"`
	Lower          float64  `yaml:"lower" json:"lower"`
	LowerInclusive bool     `yaml:"lower_inclusive" json:"lowerInclusive"`
	Upper          *float64 `yaml:"upper" json:"upper"`
	UpperInclusive bool     `yaml:"upper_inclusive" json:"upperInclusive"`
	Color          string   `yaml:"color" json:"color"`
	Icon           string   `yaml:"icon" json:"icon"`
	BeaconPattern  string   `yaml:"beacon_pattern" json:"beaconPattern"`
}

type RiskBandsConfig struct {
	Version int        `yaml:"version"`
	Unit    string     `yaml:"unit"`
	Bands   []RiskBand `yaml:"bands"`
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func LoadSourceConfig(path string) (SourceConfig, error) {
	var config SourceConfig
	if err := decodeStrictYAML(path, &config); err != nil {
		return SourceConfig{}, err
	}

	if err := config.Validate(); err != nil {
		return SourceConfig{}, fmt.Errorf("validate source config %q: %w", path, err)
	}

	return config, nil
}

func (c SourceConfig) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported source config version %d", c.Version)
	}
	if c.Timezone != "Asia/Kolkata" {
		return fmt.Errorf("timezone must be Asia/Kolkata, got %q", c.Timezone)
	}

	files := []struct {
		name     string
		got      SourceFile
		expected string
		rows     int
	}{
		{"observations", c.Observations, filepath.Join("zindi_artifacts", "Train.csv"), 48400},
		{"forecasts", c.Forecasts.SourceFile, filepath.Join("zindi_artifacts", "submission_xgb_corrected_scale_110.csv"), 2592},
		{"forecast intervals", c.ForecastIntervals, filepath.Join("zindi_artifacts", "Test.csv"), 2592},
		{"stations", c.Stations, filepath.Join("zindi_artifacts", "Stations.csv"), 4},
	}
	for _, file := range files {
		if filepath.IsAbs(file.got.Path) || filepath.Clean(file.got.Path) != filepath.Clean(file.expected) {
			return fmt.Errorf("%s path must be %q", file.name, file.expected)
		}
		if !sha256Pattern.MatchString(file.got.SHA256) {
			return fmt.Errorf("%s SHA-256 must contain exactly 64 lowercase hexadecimal characters", file.name)
		}
		if file.got.ExpectedRows != file.rows {
			return fmt.Errorf("%s expected_rows must be %d", file.name, file.rows)
		}
	}
	if c.Forecasts.SelectedScalingAlreadyApplied != 1.10 {
		return errors.New("selected forecast must declare its already-applied 1.10 scaling")
	}
	if c.Forecasts.GenerationTime != "unavailable" {
		return errors.New("forecast generation_time must remain unavailable unless supported by source evidence")
	}
	if c.OptionalEnvironmentalContext.Enabled || c.OptionalEnvironmentalContext.Status != "needs_validation" {
		return errors.New("optional environmental context must remain disabled pending validation")
	}
	if c.ClimateVerseCatalogue.DOI != "10.71646/XXOBKO" ||
		c.ClimateVerseCatalogue.Version != "1.0" ||
		c.ClimateVerseCatalogue.Stage != "planning-review" ||
		c.ClimateVerseCatalogue.DiscoveredVia != "ClimateVerse India Portal" ||
		c.ClimateVerseCatalogue.CatalogueRole != "metadata-discovery-provenance" ||
		c.ClimateVerseCatalogue.DataProvider != "Open-Meteo" {
		return errors.New("ClimateVerse catalogue provenance does not match the verified planning review")
	}

	return nil
}

func LoadRiskBands(path string) (RiskBandsConfig, error) {
	var config RiskBandsConfig
	if err := decodeStrictYAML(path, &config); err != nil {
		return RiskBandsConfig{}, err
	}
	if err := config.Validate(); err != nil {
		return RiskBandsConfig{}, fmt.Errorf("validate risk bands %q: %w", path, err)
	}

	return config, nil
}

func (c RiskBandsConfig) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported risk-bands version %d", c.Version)
	}
	if c.Unit != "ug/m3" {
		return fmt.Errorf("risk-band unit must be ug/m3, got %q", c.Unit)
	}
	if len(c.Bands) != 4 {
		return fmt.Errorf("exactly four visualization bands are required, got %d", len(c.Bands))
	}

	expectedNames := []string{"low", "elevated", "high", "very_high"}
	expectedLowers := []float64{0, 30, 60, 90}
	for i, band := range c.Bands {
		if band.Name != expectedNames[i] {
			return fmt.Errorf("band %d must be named %q", i+1, expectedNames[i])
		}
		if band.Lower != expectedLowers[i] || band.Lower < 0 {
			return fmt.Errorf("band %q lower bound must be %g and non-negative", band.Name, expectedLowers[i])
		}
		if band.LowerInclusive != (i == 0) {
			return fmt.Errorf("band %q has invalid lower-bound inclusivity", band.Name)
		}
		if i < len(c.Bands)-1 {
			expectedUpper := expectedLowers[i+1]
			if band.Upper == nil || *band.Upper != expectedUpper || !band.UpperInclusive {
				return fmt.Errorf("band %q must end inclusively at %g", band.Name, expectedUpper)
			}
		} else if band.Upper != nil {
			return fmt.Errorf("band %q must be unbounded above", band.Name)
		}
		if i > 0 {
			previous := c.Bands[i-1]
			if previous.Upper == nil || *previous.Upper != band.Lower ||
				previous.UpperInclusive == band.LowerInclusive {
				return fmt.Errorf("bands %q and %q have a gap or overlap", previous.Name, band.Name)
			}
		}
		if !colorPattern.MatchString(band.Color) {
			return fmt.Errorf("band %q must specify a #RRGGBB color", band.Name)
		}
		if strings.TrimSpace(band.Icon) == "" {
			return fmt.Errorf("band %q is missing its icon", band.Name)
		}
		if strings.TrimSpace(band.BeaconPattern) == "" {
			return fmt.Errorf("band %q is missing its beacon pattern", band.Name)
		}
	}

	return nil
}

func (c RiskBandsConfig) ClassifyPM25(value float64) (RiskBand, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return RiskBand{}, fmt.Errorf("PM2.5 must be finite and non-negative before risk classification: %v", value)
	}
	if err := c.Validate(); err != nil {
		return RiskBand{}, err
	}
	for _, band := range c.Bands {
		aboveLower := value > band.Lower || (band.LowerInclusive && value == band.Lower)
		belowUpper := band.Upper == nil || value < *band.Upper || (band.UpperInclusive && value == *band.Upper)
		if aboveLower && belowUpper {
			return band, nil
		}
	}
	return RiskBand{}, fmt.Errorf("PM2.5 value %v does not fall into a configured band", value)
}

func decodeStrictYAML(path string, destination any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open YAML config %q: %w", path, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode YAML config %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode YAML config %q: multiple documents are not supported", path)
		}
		return fmt.Errorf("decode trailing YAML config %q: %w", path, err)
	}

	return nil
}
