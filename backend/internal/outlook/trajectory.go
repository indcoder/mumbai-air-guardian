package outlook

import (
	"math"
	"sort"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
)

type TrajectoryStatus string

const (
	TrajectoryUnavailable TrajectoryStatus = "unavailable"
	TrajectoryImproving   TrajectoryStatus = "improving"
	TrajectoryStable      TrajectoryStatus = "stable"
	TrajectoryWorsening   TrajectoryStatus = "worsening"
)

type Trajectory struct {
	Status           TrajectoryStatus `json:"status"`
	SlopeUGM3PerHour *float64         `json:"slopeUgM3PerHour,omitempty"`
	PointsUsed       int              `json:"pointsUsed"`
	PartialCoverage  bool             `json:"partialCoverage"`
	LookaheadHours   int              `json:"lookaheadHours"`
}

func CalculateTrajectory(points []airdata.ForecastPoint, stationID string, anchor time.Time, lookaheadHours int, threshold float64) Trajectory {
	result := Trajectory{Status: TrajectoryUnavailable, LookaheadHours: lookaheadHours}
	if lookaheadHours <= 0 || !isFiniteNonNegative(threshold) {
		return result
	}
	end := anchor.Add(time.Duration(lookaheadHours) * time.Hour)
	selected := make([]airdata.ForecastPoint, 0, lookaheadHours)
	seen := make(map[int64]struct{}, lookaheadHours)
	for _, point := range points {
		if point.StationID != stationID ||
			point.IntervalStart.Before(anchor) ||
			!point.IntervalStart.Before(end) ||
			!isFiniteNonNegative(point.PM25) {
			continue
		}
		key := point.IntervalStart.UnixNano()
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, point)
	}
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].IntervalStart.Before(selected[j].IntervalStart)
	})
	result.PointsUsed = len(selected)
	result.PartialCoverage = len(selected) < lookaheadHours
	if len(selected) < 2 {
		return result
	}

	meanX, meanY := 0.0, 0.0
	for _, point := range selected {
		meanX += point.IntervalStart.Sub(anchor).Hours()
		meanY += point.PM25
	}
	meanX /= float64(len(selected))
	meanY /= float64(len(selected))

	covariance, variance := 0.0, 0.0
	for _, point := range selected {
		x := point.IntervalStart.Sub(anchor).Hours() - meanX
		y := point.PM25 - meanY
		covariance += x * y
		variance += x * x
	}
	if variance == 0 {
		return result
	}
	slope := covariance / variance
	result.SlopeUGM3PerHour = &slope
	switch {
	case slope > threshold:
		result.Status = TrajectoryWorsening
	case slope < -threshold:
		result.Status = TrajectoryImproving
	default:
		result.Status = TrajectoryStable
	}
	return result
}

func isFiniteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
