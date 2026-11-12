package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mumbai-air-guardian/backend/internal/activity"
	"mumbai-air-guardian/backend/internal/airdata"
)

type Activity struct {
	Language               string               `json:"language"`
	SnapshotLabel          string               `json:"snapshotLabel"`
	StationID              string               `json:"stationId"`
	StationName            string               `json:"stationName"`
	ActivityType           string               `json:"activityType"`
	Date                   string               `json:"date"`
	EarliestStart          string               `json:"earliestStart"`
	LatestEnd              string               `json:"latestEnd"`
	DurationMinutes        int                  `json:"durationMinutes"`
	PreferredStartTime     *string              `json:"preferredStartTime,omitempty"`
	RecommendedPeriod      activity.Period      `json:"recommendedPeriod"`
	Comparison             *activity.Comparison `json:"comparison,omitempty"`
	ForecastSource         airdata.SourceRecord `json:"forecastSource"`
	ForecastGenerationTime string               `json:"forecastGenerationTime"`
	ScalingAlreadyApplied  float64              `json:"scalingAlreadyApplied"`
	ForecastUncertainty    string               `json:"forecastUncertainty"`
	ForecastIntervals      []ForecastInterval   `json:"forecastIntervals"`
	ObservationQuality     ObservationQuality   `json:"observationQuality"`
}

type ForecastInterval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	PM25  float64   `json:"pm25"`
}

type ObservationQuality struct {
	FlaggedObservationCount  int       `json:"flaggedObservationCount"`
	LatestIntervalEnd        time.Time `json:"latestIntervalEnd"`
	LatestObservationFlagged bool      `json:"latestObservationFlagged"`
	ExclusionReason          string    `json:"exclusionReason"`
}

type cacheIdentity struct {
	PromptVersion string   `json:"promptVersion"`
	Model         string   `json:"model"`
	Evidence      Activity `json:"evidence"`
}

func BuildActivity(
	dataset *airdata.Dataset,
	request activity.Request,
	result activity.Result,
	language string,
) (Activity, error) {
	if dataset == nil {
		return Activity{}, errors.New("dataset is required")
	}
	if result.StationID != request.StationID || result.ActivityType != request.ActivityType ||
		result.RecommendedPeriod.CoverageHours <= 0 ||
		!result.RecommendedPeriod.Start.Before(result.RecommendedPeriod.End) {
		return Activity{}, errors.New("activity result does not match the request")
	}

	var station *airdata.Station
	for index := range dataset.Stations {
		if dataset.Stations[index].LocationID == request.StationID {
			station = &dataset.Stations[index]
			break
		}
	}
	if station == nil {
		return Activity{}, fmt.Errorf("station %q is not in the verified dataset", request.StationID)
	}

	source, found := dataset.Manifest.Sources["forecasts"]
	if !found || source.SHA256 == "" || source.SHA256 != result.SourceForecastSHA256 {
		return Activity{}, errors.New("activity result forecast provenance does not match the dataset")
	}

	points := make(map[int64]airdata.ForecastPoint, result.RecommendedPeriod.CoverageHours)
	for _, point := range dataset.Forecasts {
		if point.StationID == request.StationID {
			points[point.IntervalStart.Unix()] = point
		}
	}
	intervals := make([]ForecastInterval, 0, result.RecommendedPeriod.CoverageHours)
	for start := result.RecommendedPeriod.Start; start.Before(result.RecommendedPeriod.End); start = start.Add(time.Hour) {
		point, ok := points[start.Unix()]
		if !ok || !point.IntervalEnd.Equal(start.Add(time.Hour)) {
			return Activity{}, fmt.Errorf("forecast support is incomplete at %s", start.Format(time.RFC3339))
		}
		intervals = append(intervals, ForecastInterval{
			Start: point.IntervalStart, End: point.IntervalEnd, PM25: point.PM25,
		})
	}
	if len(intervals) != result.RecommendedPeriod.CoverageHours {
		return Activity{}, errors.New("forecast support does not match the planned duration")
	}

	quality := ObservationQuality{ExclusionReason: "Explicit source quality flag flag_high_1000=true"}
	for index := range dataset.Observations {
		observation := &dataset.Observations[index]
		if observation.StationID != request.StationID {
			continue
		}
		if observation.FlagHigh1000 {
			quality.FlaggedObservationCount++
		}
		if observation.IntervalEnd.After(quality.LatestIntervalEnd) {
			quality.LatestIntervalEnd = observation.IntervalEnd
			quality.LatestObservationFlagged = observation.FlagHigh1000
		}
	}

	return Activity{
		Language: language, SnapshotLabel: "Challenge forecast snapshot",
		StationID: request.StationID, StationName: station.Name, ActivityType: request.ActivityType,
		Date: request.Date, EarliestStart: request.EarliestStart, LatestEnd: request.LatestEnd,
		DurationMinutes: request.DurationMinutes, PreferredStartTime: request.PreferredStartTime,
		RecommendedPeriod: result.RecommendedPeriod, Comparison: result.Comparison,
		ForecastSource: source, ForecastGenerationTime: result.ForecastGeneration,
		ScalingAlreadyApplied: dataset.Manifest.SelectedForecastScaling,
		ForecastUncertainty:   "not quantified by the supplied forecast source",
		ForecastIntervals:     intervals, ObservationQuality: quality,
	}, nil
}

func CacheKey(value Activity, model, promptVersion string) (string, error) {
	encoded, err := json.Marshal(cacheIdentity{
		PromptVersion: promptVersion,
		Model:         model,
		Evidence:      value,
	})
	if err != nil {
		return "", fmt.Errorf("encode canonical explanation evidence: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
