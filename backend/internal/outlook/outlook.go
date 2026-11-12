package outlook

import (
	"errors"
	"fmt"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
	"mumbai-air-guardian/backend/internal/config"
)

type ClassifiedValue struct {
	ValuePM25 float64         `json:"valuePm25"`
	Band      config.RiskBand `json:"band"`
}

type AirOutlook struct {
	StationID          string               `json:"stationId"`
	Anchor             time.Time            `json:"anchor"`
	LatestObservation  *airdata.Observation `json:"latestObservation,omitempty"`
	ObservationBand    *config.RiskBand     `json:"observationBand,omitempty"`
	Peak               ClassifiedValue      `json:"peak"`
	Lowest             ClassifiedValue      `json:"lowest"`
	Trajectory         Trajectory           `json:"trajectory"`
	ForecastGeneration string               `json:"forecastGenerationTime"`
}

func BuildAirOutlook(
	stationID string,
	observations []airdata.Observation,
	forecasts []airdata.ForecastPoint,
	bands config.RiskBandsConfig,
	lookaheadHours int,
	slopeThreshold float64,
) (AirOutlook, error) {
	var latest *airdata.Observation
	for i := range observations {
		observation := &observations[i]
		if observation.StationID == stationID && (latest == nil || observation.IntervalEnd.After(latest.IntervalEnd)) {
			latest = observation
		}
	}
	if latest == nil {
		return AirOutlook{}, fmt.Errorf("no observations for station %q", stationID)
	}

	var anchor *airdata.ForecastPoint
	for i := range forecasts {
		point := &forecasts[i]
		if point.StationID == stationID && !point.IntervalStart.Before(latest.IntervalEnd) &&
			(anchor == nil || point.IntervalStart.Before(anchor.IntervalStart)) {
			anchor = point
		}
	}
	if anchor == nil {
		return AirOutlook{}, fmt.Errorf("no post-observation forecast for station %q", stationID)
	}

	forward := make([]airdata.ForecastPoint, 0, len(forecasts))
	var peak, lowest *airdata.ForecastPoint
	for i := range forecasts {
		point := &forecasts[i]
		if point.StationID != stationID || point.IntervalStart.Before(latest.IntervalEnd) {
			continue
		}
		forward = append(forward, *point)
		if peak == nil || point.PM25 > peak.PM25 ||
			(point.PM25 == peak.PM25 && point.IntervalStart.Before(peak.IntervalStart)) {
			peak = point
		}
		if lowest == nil || point.PM25 < lowest.PM25 ||
			(point.PM25 == lowest.PM25 && point.IntervalStart.Before(lowest.IntervalStart)) {
			lowest = point
		}
	}
	if peak == nil || lowest == nil {
		return AirOutlook{}, errors.New("no valid post-observation forecast values")
	}
	var observationBand *config.RiskBand
	if latest.SummaryEligible() {
		classified, err := bands.ClassifyPM25(latest.PM25)
		if err != nil {
			return AirOutlook{}, fmt.Errorf("classify latest observation: %w", err)
		}
		observationBand = &classified
	}
	peakBand, err := bands.ClassifyPM25(peak.PM25)
	if err != nil {
		return AirOutlook{}, fmt.Errorf("classify forecast peak: %w", err)
	}
	lowestBand, err := bands.ClassifyPM25(lowest.PM25)
	if err != nil {
		return AirOutlook{}, fmt.Errorf("classify lowest forecast: %w", err)
	}
	trajectory := CalculateTrajectory(forward, stationID, anchor.IntervalStart, lookaheadHours, slopeThreshold)
	return AirOutlook{
		StationID: stationID, Anchor: anchor.IntervalStart, LatestObservation: latest,
		ObservationBand: observationBand,
		Peak:            ClassifiedValue{ValuePM25: peak.PM25, Band: peakBand},
		Lowest:          ClassifiedValue{ValuePM25: lowest.PM25, Band: lowestBand},
		Trajectory:      trajectory, ForecastGeneration: "unavailable",
	}, nil
}
