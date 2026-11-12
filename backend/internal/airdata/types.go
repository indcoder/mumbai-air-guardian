package airdata

import (
	"time"

	"mumbai-air-guardian/backend/internal/config"
)

type SourceKind string

const (
	ObservedSource  SourceKind = "observed"
	ForecastSource  SourceKind = "forecast"
	ReferenceSource SourceKind = "reference"
)

type SourceRecord struct {
	Kind   SourceKind `json:"kind"`
	Path   string     `json:"path"`
	SHA256 string     `json:"sha256"`
	Rows   int        `json:"rows"`
}

type Manifest struct {
	Timezone                 string                       `json:"timezone"`
	ForecastGeneratedAt      *time.Time                   `json:"forecastGeneratedAt"`
	ForecastGenerationStatus string                       `json:"forecastGenerationStatus"`
	SelectedForecastScaling  float64                      `json:"selectedForecastScalingAlreadyApplied"`
	Sources                  map[string]SourceRecord      `json:"sources"`
	ClimateVerse             config.ClimateVerseCatalogue `json:"climateVerseCatalogue"`
}

type Station struct {
	LocationID string  `json:"locationId"`
	Name       string  `json:"name"`
	SensorID   string  `json:"sensorId"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Timezone   string  `json:"timezone"`
}

type Observation struct {
	StationID              string     `json:"stationId"`
	SensorID               string     `json:"sensorId"`
	IntervalStart          time.Time  `json:"intervalStart"`
	IntervalEnd            time.Time  `json:"intervalEnd"`
	SourceTimestamp        time.Time  `json:"sourceTimestamp"`
	AvailableAtSourceMTime time.Time  `json:"availableAtSourceMtime"`
	PM25                   float64    `json:"pm25"`
	Mean                   float64    `json:"mean"`
	Count                  int        `json:"count"`
	StationName            string     `json:"stationName"`
	SeriesID               string     `json:"seriesId"`
	FlagHigh1000           bool       `json:"flagHigh1000"`
	SourceKind             SourceKind `json:"sourceKind"`
}

func (o Observation) SummaryEligible() bool {
	return !o.FlagHigh1000
}

func (o Observation) ExclusionReason() string {
	if o.FlagHigh1000 {
		return "Explicit source quality flag flag_high_1000=true"
	}
	return ""
}

type ForecastPoint struct {
	ID            string     `json:"id"`
	StationID     string     `json:"stationId"`
	SensorID      string     `json:"sensorId"`
	IntervalStart time.Time  `json:"intervalStart"`
	IntervalEnd   time.Time  `json:"intervalEnd"`
	PM25          float64    `json:"pm25"`
	SourceKind    SourceKind `json:"sourceKind"`
	SourceRow     int        `json:"sourceRow"`
}

type Dataset struct {
	Stations     []Station
	Observations []Observation
	Forecasts    []ForecastPoint
	RiskBands    config.RiskBandsConfig
	Manifest     Manifest
}

func (d *Dataset) FlaggedObservationCount() int {
	count := 0
	for _, observation := range d.Observations {
		if observation.FlagHigh1000 {
			count++
		}
	}
	return count
}
