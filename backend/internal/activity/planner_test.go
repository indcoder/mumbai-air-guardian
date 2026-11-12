package activity

import (
	"encoding/json"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
)

func testProjectRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func TestMaladActivityRegressionWithPreferredPeriod(t *testing.T) {
	dataset, err := airdata.LoadFromRoot(testProjectRoot(t))
	if err != nil {
		t.Fatalf("load source-backed regression data: %v", err)
	}
	planner, err := NewPlanner(dataset)
	if err != nil {
		t.Fatal(err)
	}
	preferredStart := "08:00"
	result, err := planner.Plan(Request{
		StationID: "11611", ActivityType: "outdoor_activity", Date: "2026-11-12",
		EarliestStart: "06:00", LatestEnd: "20:00", DurationMinutes: 120,
		PreferredStartTime: &preferredStart,
	})
	if err != nil {
		t.Fatalf("plan from the selected forecast: %v", err)
	}
	if result.Comparison == nil {
		t.Fatal("preferred interval was not compared")
	}
	assertNear(t, result.Comparison.PreferredPeriod.MeanPM25, 86.1471275, 1e-7)
	assertNear(t, result.RecommendedPeriod.MeanPM25, 52.5801715, 1e-7)
	assertNear(t, result.Comparison.DifferencePM25, -33.566956, 1e-7)
	if result.RecommendedPeriod.Start.Format("15:04") != "15:00" ||
		result.RecommendedPeriod.End.Format("15:04") != "17:00" {
		t.Fatalf("recommended period = %s-%s, want 15:00-17:00",
			result.RecommendedPeriod.Start, result.RecommendedPeriod.End)
	}
}

func TestNoPreferredTimeOmitsEveryComparisonFieldFromAPIJSON(t *testing.T) {
	dataset, err := airdata.LoadFromRoot(testProjectRoot(t))
	if err != nil {
		t.Fatalf("load source-backed forecast data: %v", err)
	}
	planner, err := NewPlanner(dataset)
	if err != nil {
		t.Fatal(err)
	}
	result, err := planner.Plan(Request{
		StationID: "11611", ActivityType: "outdoor_activity", Date: "2026-11-12",
		EarliestStart: "06:00", LatestEnd: "20:00", DurationMinutes: 120,
	})
	if err != nil {
		t.Fatalf("plan without a preferred time: %v", err)
	}
	if result.Comparison != nil {
		t.Fatalf("comparison unexpectedly exists: %+v", result.Comparison)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "comparison") ||
		strings.Contains(string(encoded), "preferredPeriod") ||
		strings.Contains(string(encoded), "differencePm25") {
		t.Fatalf("no-preference API result contains comparison fields: %s", encoded)
	}
}

func TestPlannerRejectsMissingCoverageAndInvalidResolution(t *testing.T) {
	planner := fixturePlanner(t, map[int64]float64{hourKey("2026-11-12T06:00:00+05:30"): 10})
	base := Request{
		StationID: "11611", ActivityType: "run", Date: "2026-11-12",
		EarliestStart: "06:00", LatestEnd: "08:00", DurationMinutes: 120,
	}
	if _, err := planner.Plan(base); err != ErrInsufficientCoverage {
		t.Fatalf("incomplete candidate error = %v, want insufficient coverage", err)
	}
	base.DurationMinutes = 90
	if _, err := planner.Plan(base); err == nil || !strings.Contains(err.Error(), "whole-hour") {
		t.Fatalf("sub-hour precision was not rejected: %v", err)
	}
}

func TestPlannerUsesEarliestStartForTiesWithinTolerance(t *testing.T) {
	series := map[int64]float64{
		hourKey("2026-11-12T06:00:00+05:30"): 20,
		hourKey("2026-11-12T07:00:00+05:30"): 10,
		hourKey("2026-11-12T08:00:00+05:30"): 10 + 5e-10,
	}
	planner := fixturePlanner(t, series)
	result, err := planner.Plan(Request{
		StationID: "11611", ActivityType: "walk", Date: "2026-11-12",
		EarliestStart: "06:00", LatestEnd: "09:00", DurationMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RecommendedPeriod.Start.Hour() != 7 {
		t.Fatalf("tie winner starts at %s, want earliest tied start 07:00", result.RecommendedPeriod.Start)
	}
}

func TestPlannerPreferredWindowMustBeCoveredAndWithinAllowedRange(t *testing.T) {
	planner := fixturePlanner(t, map[int64]float64{
		hourKey("2026-11-12T06:00:00+05:30"): 10,
		hourKey("2026-11-12T07:00:00+05:30"): 9,
	})
	preferred := "07:00"
	request := Request{
		StationID: "11611", ActivityType: "walk", Date: "2026-11-12",
		EarliestStart: "06:00", LatestEnd: "09:00", DurationMinutes: 120,
		PreferredStartTime: &preferred,
	}
	if _, err := planner.Plan(request); err == nil || !strings.Contains(err.Error(), "preferred period") {
		t.Fatalf("incomplete preferred interval was not rejected: %v", err)
	}
	preferred = "08:00"
	if _, err := planner.Plan(request); err == nil || !strings.Contains(err.Error(), "fit within") {
		t.Fatalf("out-of-range preferred interval was not rejected: %v", err)
	}
}

func fixturePlanner(t *testing.T, series map[int64]float64) *Planner {
	t.Helper()
	location := time.FixedZone("IST", 19800)
	stations := []airdata.Station{{LocationID: "11611", Timezone: "Asia/Kolkata"}}
	observations := []airdata.Observation{{
		StationID:   "11611",
		IntervalEnd: time.Date(2026, 11, 10, 9, 0, 0, 0, location),
	}}
	points := make([]airdata.ForecastPoint, 0, len(series))
	for unix, value := range series {
		points = append(points, airdata.ForecastPoint{
			StationID: "11611", IntervalStart: time.Unix(unix, 0).In(location),
			IntervalEnd: time.Unix(unix, 0).In(location).Add(time.Hour), PM25: value,
		})
	}
	dataset := &airdata.Dataset{
		Stations: stations, Observations: observations, Forecasts: points,
		Manifest: airdata.Manifest{
			Timezone: "Asia/Kolkata",
			Sources: map[string]airdata.SourceRecord{
				"forecasts": {SHA256: "fixture"},
			},
		},
	}
	planner, err := NewPlanner(dataset)
	if err != nil {
		t.Fatal(err)
	}
	return planner
}

func hourKey(value string) int64 {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed.Unix()
}

func assertNear(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("got %.10f, want %.10f (tolerance %.10f)", got, want, tolerance)
	}
}
