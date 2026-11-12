package outlook

import (
	"math"
	"testing"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
)

func TestTrajectoryUsesActualElapsedTimeAndMarksMissingHours(t *testing.T) {
	anchor := time.Date(2026, 11, 12, 9, 0, 0, 0, time.FixedZone("IST", 19800))
	points := []airdata.ForecastPoint{
		{StationID: "11611", IntervalStart: anchor, PM25: 10},
		{StationID: "11611", IntervalStart: anchor.Add(2 * time.Hour), PM25: 14},
	}
	result := CalculateTrajectory(points, "11611", anchor, 6, 2)
	if result.Status != TrajectoryStable || result.SlopeUGM3PerHour == nil ||
		math.Abs(*result.SlopeUGM3PerHour-2) > 1e-12 {
		t.Fatalf("trajectory compressed away the missing hour: %+v", result)
	}
	if result.PointsUsed != 2 || !result.PartialCoverage {
		t.Fatalf("partial coverage was not reported: %+v", result)
	}
}

func TestTrajectoryDoesNotExtendLookaheadAndRequiresTwoValidPoints(t *testing.T) {
	anchor := time.Date(2026, 11, 12, 9, 0, 0, 0, time.FixedZone("IST", 19800))
	points := []airdata.ForecastPoint{
		{StationID: "11611", IntervalStart: anchor, PM25: 20},
		{StationID: "11611", IntervalStart: anchor.Add(6 * time.Hour), PM25: 100},
	}
	result := CalculateTrajectory(points, "11611", anchor, 6, 2)
	if result.Status != TrajectoryUnavailable || result.PointsUsed != 1 ||
		result.SlopeUGM3PerHour != nil {
		t.Fatalf("point at the exclusive lookahead end was used: %+v", result)
	}
}

func TestTrajectoryClassifiesPositiveNegativeAndThresholdSlopes(t *testing.T) {
	anchor := time.Date(2026, 11, 12, 9, 0, 0, 0, time.FixedZone("IST", 19800))
	tests := []struct {
		name   string
		values []float64
		want   TrajectoryStatus
	}{
		{"worsening", []float64{10, 13}, TrajectoryWorsening},
		{"improving", []float64{13, 10}, TrajectoryImproving},
		{"exact threshold stable", []float64{10, 12}, TrajectoryStable},
		{"constant stable", []float64{10, 10}, TrajectoryStable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			points := []airdata.ForecastPoint{
				{StationID: "11611", IntervalStart: anchor, PM25: test.values[0]},
				{StationID: "11611", IntervalStart: anchor.Add(time.Hour), PM25: test.values[1]},
			}
			result := CalculateTrajectory(points, "11611", anchor, 6, 2)
			if result.Status != test.want {
				t.Fatalf("status = %q, want %q (%+v)", result.Status, test.want, result)
			}
		})
	}
}
