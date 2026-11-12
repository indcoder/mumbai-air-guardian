package airdata

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mumbai-air-guardian/backend/internal/config"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func TestLoadAuthoritativeDatasetAndPreserveQualityFlags(t *testing.T) {
	dataset, err := LoadFromRoot(projectRoot(t))
	if err != nil {
		t.Fatalf("load immutable challenge inputs: %v", err)
	}
	if len(dataset.Stations) != 4 {
		t.Fatalf("got %d stations, want 4", len(dataset.Stations))
	}
	if len(dataset.Observations) != 48400 {
		t.Fatalf("got %d observations, want 48400", len(dataset.Observations))
	}
	if len(dataset.Forecasts) != 2592 {
		t.Fatalf("got %d forecasts, want 2592", len(dataset.Forecasts))
	}
	if got := dataset.FlaggedObservationCount(); got != 19 {
		t.Fatalf("got %d flagged observations, want 19", got)
	}
	for _, observation := range dataset.Observations {
		if observation.FlagHigh1000 {
			if observation.SummaryEligible() {
				t.Fatal("explicitly flagged observation entered ordinary summaries")
			}
			if observation.ExclusionReason() == "" {
				t.Fatal("flagged observation is missing its diagnostic reason")
			}
		}
		if observation.SourceKind != ObservedSource {
			t.Fatalf("observation source kind = %q", observation.SourceKind)
		}
	}
	if dataset.Manifest.ForecastGeneratedAt != nil ||
		dataset.Manifest.ForecastGenerationStatus != "unavailable" {
		t.Fatal("unrecorded forecast generation time was fabricated")
	}
	if dataset.Manifest.SelectedForecastScaling != 1.10 {
		t.Fatalf("selected scaling = %v, want 1.10 already applied", dataset.Manifest.SelectedForecastScaling)
	}
	if dataset.Manifest.ClimateVerse.DiscoveredVia != "ClimateVerse India Portal" ||
		dataset.Manifest.ClimateVerse.DataProvider != "Open-Meteo" {
		t.Fatal("catalogue and weather-provider roles were conflated")
	}
}

func TestForecastIntervalsUseTestColumnsAndKeepObservedBoundarySeparate(t *testing.T) {
	dataset, err := LoadFromRoot(projectRoot(t))
	if err != nil {
		t.Fatalf("load immutable challenge inputs: %v", err)
	}

	var sample *ForecastPoint
	var latestObservation time.Time
	var firstForecast *ForecastPoint
	for i := range dataset.Forecasts {
		point := &dataset.Forecasts[i]
		if point.ID == "MUM_11611_20261027T0100_IST" {
			sample = point
		}
		if point.StationID == "11611" && point.IntervalStart.Equal(time.Date(2026, 11, 10, 9, 0, 0, 0, time.FixedZone("IST", 5*60*60+30*60))) {
			firstForecast = point
		}
	}
	for _, observation := range dataset.Observations {
		if observation.StationID == "11611" && observation.IntervalEnd.After(latestObservation) {
			latestObservation = observation.IntervalEnd
		}
	}
	if sample == nil {
		t.Fatal("timestamp-boundary sample ID was not loaded")
	}
	if sample.IntervalStart.Hour() != 0 || sample.IntervalEnd.Hour() != 1 ||
		sample.IntervalStart.Day() != 27 || sample.IntervalStart.Month() != time.October {
		t.Fatalf("sample ID interval = %s-%s, want 2026-10-27 00:00-01:00 IST", sample.IntervalStart, sample.IntervalEnd)
	}
	if latestObservation.Hour() != 9 || latestObservation.Day() != 10 ||
		latestObservation.Month() != time.November || latestObservation.Year() != 2026 {
		t.Fatalf("latest observation end = %s, want 2026-11-10 09:00 IST", latestObservation)
	}
	if firstForecast == nil {
		t.Fatal("first post-observation forecast was not found")
	}
	if !firstForecast.IntervalStart.Equal(latestObservation) ||
		firstForecast.IntervalStart.Hour() != 9 || firstForecast.IntervalEnd.Hour() != 10 {
		t.Fatalf("observed/forecast boundary is not 2026-11-10 09:00 IST: observed end %s, forecast %s-%s",
			latestObservation, firstForecast.IntervalStart, firstForecast.IntervalEnd)
	}
	if firstForecast.PM25 != 74.129998 {
		t.Fatalf("selected prediction changed or was scaled again: got %.9f, want 74.129998", firstForecast.PM25)
	}
}

func TestForecastSeriesAreChronologicalAndRetainTheChallengeGap(t *testing.T) {
	dataset, err := LoadFromRoot(projectRoot(t))
	if err != nil {
		t.Fatalf("load immutable challenge inputs: %v", err)
	}
	var previous *ForecastPoint
	foundGap := false
	foundChallengeGap := false
	for i := range dataset.Forecasts {
		point := &dataset.Forecasts[i]
		if previous != nil && previous.StationID == point.StationID {
			if !point.IntervalStart.After(previous.IntervalStart) {
				t.Fatalf("forecast intervals are not strictly ordered for station %s", point.StationID)
			}
			if point.IntervalStart.Sub(previous.IntervalEnd) > time.Hour {
				foundGap = true
				if point.StationID == "11611" &&
					previous.IntervalEnd.Equal(time.Date(2026, 10, 24, 0, 0, 0, 0, previous.IntervalEnd.Location())) &&
					point.IntervalStart.Equal(time.Date(2026, 10, 27, 0, 0, 0, 0, point.IntervalStart.Location())) {
					foundChallengeGap = true
				}
			}
		}
		previous = point
	}
	if !foundGap {
		t.Fatal("expected historical challenge forecast gap was filled or lost")
	}
	if !foundChallengeGap {
		t.Fatal("the 24-26 October forecast gap was not preserved")
	}
}

func TestHashMismatchRejectsChangedSource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "zindi_artifacts", "Train.csv")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := config.SourceFile{
		Path:   filepath.Join("zindi_artifacts", "Train.csv"),
		SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	if got, _, err := readVerifiedFile(root, source); err == nil || got != nil {
		t.Fatal("changed registered source was accepted")
	}
}
