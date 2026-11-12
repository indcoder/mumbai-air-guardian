package config

import (
	"math"
	"testing"
)

func validRiskBands() RiskBandsConfig {
	upper30, upper60, upper90 := 30.0, 60.0, 90.0
	return RiskBandsConfig{
		Version: 1,
		Unit:    "ug/m3",
		Bands: []RiskBand{
			{Name: "low", Lower: 0, LowerInclusive: true, Upper: &upper30, UpperInclusive: true, Color: "#2D6A4F", Icon: "check", BeaconPattern: "steady"},
			{Name: "elevated", Lower: 30, LowerInclusive: false, Upper: &upper60, UpperInclusive: true, Color: "#9A6700", Icon: "alert", BeaconPattern: "single"},
			{Name: "high", Lower: 60, LowerInclusive: false, Upper: &upper90, UpperInclusive: true, Color: "#B54708", Icon: "warning", BeaconPattern: "double"},
			{Name: "very_high", Lower: 90, LowerInclusive: false, Color: "#A61B29", Icon: "octagon", BeaconPattern: "triple"},
		},
	}
}

func TestRiskBandsAcceptApprovedBoundaries(t *testing.T) {
	if err := validRiskBands().Validate(); err != nil {
		t.Fatalf("valid bands rejected: %v", err)
	}
}

func TestRiskBandsRejectInvalidBoundariesAndMissingPresentation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RiskBandsConfig)
	}{
		{"negative lower bound", func(c *RiskBandsConfig) { c.Bands[0].Lower = -1 }},
		{"gap", func(c *RiskBandsConfig) { c.Bands[1].Lower = 31 }},
		{"overlap", func(c *RiskBandsConfig) { c.Bands[1].LowerInclusive = true }},
		{"invalid ordering", func(c *RiskBandsConfig) { c.Bands[2].Lower = 30 }},
		{"missing color", func(c *RiskBandsConfig) { c.Bands[0].Color = "" }},
		{"missing icon", func(c *RiskBandsConfig) { c.Bands[1].Icon = "" }},
		{"missing beacon pattern", func(c *RiskBandsConfig) { c.Bands[2].BeaconPattern = "" }},
		{"finite final upper", func(c *RiskBandsConfig) { upper := 120.0; c.Bands[3].Upper = &upper }},
		{"negative infinity lower", func(c *RiskBandsConfig) { c.Bands[0].Lower = math.Inf(-1) }},
		{"NaN lower", func(c *RiskBandsConfig) { c.Bands[0].Lower = math.NaN() }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validRiskBands()
			test.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("invalid risk-band configuration was accepted")
			}
		})
	}
}

func TestRiskBandsClassifyExactBoundariesAndRejectInvalidMeasurements(t *testing.T) {
	config := validRiskBands()
	tests := []struct {
		value float64
		want  string
	}{
		{0, "low"},
		{30, "low"},
		{30.000001, "elevated"},
		{60, "elevated"},
		{60.000001, "high"},
		{90, "high"},
		{90.000001, "very_high"},
	}
	for _, test := range tests {
		got, err := config.ClassifyPM25(test.value)
		if err != nil {
			t.Fatalf("classify %g: %v", test.value, err)
		}
		if got.Name != test.want {
			t.Errorf("classify %g = %q, want %q", test.value, got.Name, test.want)
		}
	}
	for _, value := range []float64{-0.001, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := config.ClassifyPM25(value); err == nil {
			t.Errorf("invalid PM2.5 %v entered risk classification", value)
		}
	}
}
