package outlook

import (
	"testing"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
	"mumbai-air-guardian/backend/internal/config"
)

func TestFlaggedLatestObservationRemainsDiagnosticWithoutRiskBand(t *testing.T) {
	location := time.FixedZone("IST", 5*60*60+30*60)
	latestEnd := time.Date(2026, 11, 10, 9, 0, 0, 0, location)
	dataset := &airdata.Dataset{
		Observations: []airdata.Observation{{
			StationID: "11611", IntervalEnd: latestEnd, PM25: 120,
			FlagHigh1000: true, SourceKind: airdata.ObservedSource,
		}},
		Forecasts: []airdata.ForecastPoint{
			{StationID: "11611", IntervalStart: latestEnd, IntervalEnd: latestEnd.Add(time.Hour), PM25: 20},
			{StationID: "11611", IntervalStart: latestEnd.Add(time.Hour), IntervalEnd: latestEnd.Add(2 * time.Hour), PM25: 22},
		},
	}
	bands := config.RiskBandsConfig{
		Version: 1, Unit: "ug/m3",
		Bands: []config.RiskBand{
			{Name: "low", Lower: 0, LowerInclusive: true, Upper: floatPointer(30), UpperInclusive: true, Color: "#2D6A4F", Icon: "circle-check", BeaconPattern: "steady"},
			{Name: "elevated", Lower: 30, Upper: floatPointer(60), UpperInclusive: true, Color: "#9A6700", Icon: "triangle-alert", BeaconPattern: "single-pulse"},
			{Name: "high", Lower: 60, Upper: floatPointer(90), UpperInclusive: true, Color: "#B54708", Icon: "triangle-exclamation", BeaconPattern: "double-pulse"},
			{Name: "very_high", Lower: 90, Color: "#A61B29", Icon: "octagon-alert", BeaconPattern: "triple-pulse"},
		},
	}

	result, err := BuildAirOutlook("11611", dataset.Observations, dataset.Forecasts, bands, 6, 2)
	if err != nil {
		t.Fatalf("build outlook: %v", err)
	}
	if result.LatestObservation == nil || !result.LatestObservation.FlagHigh1000 {
		t.Fatal("source-flagged latest observation was not retained")
	}
	if result.LatestObservation.PM25 != 120 {
		t.Fatalf("raw flagged value = %v, want 120", result.LatestObservation.PM25)
	}
	if result.ObservationBand != nil {
		t.Fatalf("flagged observation received an ordinary risk band: %+v", result.ObservationBand)
	}
}

func floatPointer(value float64) *float64 {
	return &value
}
