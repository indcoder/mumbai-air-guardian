package activity

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
)

const meanTieTolerance = 1e-9

var (
	ErrInvalidRequest       = errors.New("invalid activity request")
	ErrInsufficientCoverage = errors.New("insufficient contiguous forecast coverage")
	ErrUnknownStation       = errors.New("unknown station")
)

type Request struct {
	StationID          string  `json:"stationId"`
	ActivityType       string  `json:"activityType"`
	Date               string  `json:"date"`
	EarliestStart      string  `json:"earliestStart"`
	LatestEnd          string  `json:"latestEnd"`
	DurationMinutes    int     `json:"durationMinutes"`
	PreferredStartTime *string `json:"preferredStartTime,omitempty"`
}

type Period struct {
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	MeanPM25      float64   `json:"meanPm25"`
	CoverageHours int       `json:"coverageHours"`
}

type Comparison struct {
	PreferredPeriod Period  `json:"preferredPeriod"`
	DifferencePM25  float64 `json:"differencePm25"`
}

type Result struct {
	StationID            string      `json:"stationId"`
	ActivityType         string      `json:"activityType"`
	RecommendedPeriod    Period      `json:"recommendedPeriod"`
	Comparison           *Comparison `json:"comparison,omitempty"`
	ResolutionMinutes    int         `json:"resolutionMinutes"`
	SourceForecastSHA256 string      `json:"sourceForecastSha256"`
	ForecastGeneration   string      `json:"forecastGenerationTime"`
}

type Planner struct {
	location          *time.Location
	stationIDs        map[string]struct{}
	forecastByStation map[string]map[int64]float64
	sourceHash        string
}

type forecastWindow struct {
	start          time.Time
	prefixSums     []float64
	prefixCoverage []int
}

func NewPlanner(dataset *airdata.Dataset) (*Planner, error) {
	if dataset == nil {
		return nil, errors.New("dataset is required")
	}
	location, err := time.LoadLocation(dataset.Manifest.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load planner timezone %q: %w", dataset.Manifest.Timezone, err)
	}

	planner := &Planner{
		location:          location,
		stationIDs:        make(map[string]struct{}, len(dataset.Stations)),
		forecastByStation: make(map[string]map[int64]float64, len(dataset.Stations)),
		sourceHash:        dataset.Manifest.Sources["forecasts"].SHA256,
	}
	observedThrough := make(map[string]time.Time, len(dataset.Stations))
	for _, station := range dataset.Stations {
		planner.stationIDs[station.LocationID] = struct{}{}
		planner.forecastByStation[station.LocationID] = make(map[int64]float64)
	}
	for _, observation := range dataset.Observations {
		if observation.IntervalEnd.After(observedThrough[observation.StationID]) {
			observedThrough[observation.StationID] = observation.IntervalEnd
		}
	}
	for _, point := range dataset.Forecasts {
		if _, known := planner.stationIDs[point.StationID]; !known {
			return nil, fmt.Errorf("forecast references unknown station %q", point.StationID)
		}
		if point.IntervalStart.Before(observedThrough[point.StationID]) {
			continue
		}
		hour := point.IntervalStart.Unix()
		if _, duplicate := planner.forecastByStation[point.StationID][hour]; duplicate {
			return nil, fmt.Errorf("duplicate forward forecast interval for station %q at %s",
				point.StationID, point.IntervalStart.Format(time.RFC3339))
		}
		planner.forecastByStation[point.StationID][hour] = point.PM25
	}
	return planner, nil
}

func (p *Planner) Plan(request Request) (Result, error) {
	if _, ok := p.stationIDs[request.StationID]; !ok {
		return Result{}, fmt.Errorf("%w: %q", ErrUnknownStation, request.StationID)
	}
	request.ActivityType = strings.TrimSpace(request.ActivityType)
	if request.ActivityType == "" || len(request.ActivityType) > 64 {
		return Result{}, fmt.Errorf("%w: activityType must contain 1 to 64 characters", ErrInvalidRequest)
	}
	if request.DurationMinutes <= 0 || request.DurationMinutes%60 != 0 || request.DurationMinutes > 24*60 {
		return Result{}, fmt.Errorf("%w: durationMinutes must be a positive whole-hour duration", ErrInvalidRequest)
	}
	duration := time.Duration(request.DurationMinutes) * time.Minute
	date, err := time.Parse("2006-01-02", request.Date)
	if err != nil {
		return Result{}, fmt.Errorf("%w: date must use YYYY-MM-DD", ErrInvalidRequest)
	}
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, p.location)
	earliest, err := parseClockTime(dayStart, request.EarliestStart, false)
	if err != nil {
		return Result{}, fmt.Errorf("%w: earliestStart: %v", ErrInvalidRequest, err)
	}
	latest, err := parseClockTime(dayStart, request.LatestEnd, true)
	if err != nil {
		return Result{}, fmt.Errorf("%w: latestEnd: %v", ErrInvalidRequest, err)
	}
	if !earliest.Before(latest) {
		return Result{}, fmt.Errorf("%w: allowed interval must have a positive length", ErrInvalidRequest)
	}
	if duration > latest.Sub(earliest) {
		return Result{}, fmt.Errorf("%w: duration does not fit the allowed interval", ErrInvalidRequest)
	}

	series := p.forecastByStation[request.StationID]
	window := newForecastWindow(series, earliest, latest)
	durationHours := int(duration / time.Hour)
	recommended, ok := window.lowestCompleteWindow(durationHours)
	if !ok {
		return Result{}, ErrInsufficientCoverage
	}
	result := Result{
		StationID: request.StationID, ActivityType: request.ActivityType,
		RecommendedPeriod: recommended, ResolutionMinutes: 60,
		SourceForecastSHA256: p.sourceHash, ForecastGeneration: "unavailable",
	}
	if request.PreferredStartTime == nil {
		return result, nil
	}

	preferredStart, err := parseClockTime(dayStart, *request.PreferredStartTime, false)
	if err != nil {
		return Result{}, fmt.Errorf("%w: preferredStartTime: %v", ErrInvalidRequest, err)
	}
	preferredEnd := preferredStart.Add(duration)
	if preferredStart.Before(earliest) || preferredEnd.After(latest) {
		return Result{}, fmt.Errorf("%w: preferred period must fit within the allowed interval", ErrInvalidRequest)
	}
	preferred, ok := window.period(preferredStart, durationHours)
	if !ok {
		return Result{}, fmt.Errorf("%w: preferred period is missing complete contiguous hourly coverage", ErrInsufficientCoverage)
	}
	difference := recommended.MeanPM25 - preferred.MeanPM25
	result.Comparison = &Comparison{
		PreferredPeriod: preferred,
		DifferencePM25:  difference,
	}
	return result, nil
}

func newForecastWindow(series map[int64]float64, start, end time.Time) forecastWindow {
	hours := int(end.Sub(start) / time.Hour)
	window := forecastWindow{
		start:          start,
		prefixSums:     make([]float64, hours+1),
		prefixCoverage: make([]int, hours+1),
	}
	for hour := 0; hour < hours; hour++ {
		window.prefixSums[hour+1] = window.prefixSums[hour]
		window.prefixCoverage[hour+1] = window.prefixCoverage[hour]
		if value, ok := series[start.Add(time.Duration(hour)*time.Hour).Unix()]; ok && isFiniteNonNegative(value) {
			window.prefixSums[hour+1] += value
			window.prefixCoverage[hour+1]++
		}
	}
	return window
}

func (w forecastWindow) lowestCompleteWindow(durationHours int) (Period, bool) {
	var best Period
	found := false
	for startIndex := 0; startIndex+durationHours < len(w.prefixSums); startIndex++ {
		candidate, complete := w.periodAt(startIndex, durationHours)
		if !complete {
			continue
		}
		if !found || candidate.MeanPM25 < best.MeanPM25-meanTieTolerance ||
			(math.Abs(candidate.MeanPM25-best.MeanPM25) <= meanTieTolerance && candidate.Start.Before(best.Start)) {
			best = candidate
			found = true
		}
	}
	return best, found
}

func (w forecastWindow) period(start time.Time, durationHours int) (Period, bool) {
	offset := start.Sub(w.start)
	if offset < 0 || offset%time.Hour != 0 {
		return Period{}, false
	}
	return w.periodAt(int(offset/time.Hour), durationHours)
}

func (w forecastWindow) periodAt(startIndex, durationHours int) (Period, bool) {
	endIndex := startIndex + durationHours
	if startIndex < 0 || durationHours <= 0 || endIndex >= len(w.prefixSums) ||
		w.prefixCoverage[endIndex]-w.prefixCoverage[startIndex] != durationHours {
		return Period{}, false
	}
	sum := w.prefixSums[endIndex] - w.prefixSums[startIndex]
	return Period{
		Start:    w.start.Add(time.Duration(startIndex) * time.Hour),
		End:      w.start.Add(time.Duration(endIndex) * time.Hour),
		MeanPM25: sum / float64(durationHours), CoverageHours: durationHours,
	}, true
}

func parseClockTime(dayStart time.Time, value string, allowEndOfDay bool) (time.Time, error) {
	if allowEndOfDay && value == "24:00" {
		return dayStart.Add(24 * time.Hour), nil
	}
	clock, err := time.Parse("15:04", value)
	if err != nil || len(value) != 5 || clock.Minute() != 0 {
		return time.Time{}, errors.New("time must be a whole-hour HH:00 value")
	}
	return time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(),
		clock.Hour(), clock.Minute(), 0, 0, dayStart.Location()), nil
}

func isFiniteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
