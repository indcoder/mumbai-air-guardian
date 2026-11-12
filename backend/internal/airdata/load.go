package airdata

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"mumbai-air-guardian/backend/internal/config"
	_ "time/tzdata"
)

var expectedStations = map[string]struct {
	name   string
	sensor string
}{
	"11611":   {name: "Malad West, Mumbai - IITM", sensor: "12236426"},
	"3409511": {name: "Shivaji Nagar, Mumbai - BMC", sensor: "12238576"},
	"11606":   {name: "Borivali East, Mumbai - IITM", sensor: "12236418"},
	"3409510": {name: "Byculla, Mumbai - BMC", sensor: "12238567"},
}

var stationOrder = []string{"11611", "3409511", "11606", "3409510"}

func LoadFromRoot(root string) (*Dataset, error) {
	sourceConfigPath := filepath.Join(root, filepath.FromSlash(config.SourceConfigRelativePath))
	sourceConfig, err := config.LoadSourceConfig(sourceConfigPath)
	if err != nil {
		return nil, err
	}

	riskBandsPath := filepath.Join(root, filepath.FromSlash(config.RiskBandsRelativePath))
	riskBands, err := config.LoadRiskBands(riskBandsPath)
	if err != nil {
		return nil, err
	}

	stationBytes, stationHash, err := readVerifiedFile(root, sourceConfig.Stations)
	if err != nil {
		return nil, err
	}
	stationRows, err := parseStations(stationBytes, sourceConfig.Stations.Path)
	if err != nil {
		return nil, err
	}

	observationBytes, observationHash, err := readVerifiedFile(root, sourceConfig.Observations)
	if err != nil {
		return nil, err
	}
	observations, err := parseObservations(observationBytes, sourceConfig.Observations.Path, stationRows)
	if err != nil {
		return nil, err
	}
	if len(observations) != sourceConfig.Observations.ExpectedRows {
		return nil, fmt.Errorf("observation row count %d does not match expected %d", len(observations), sourceConfig.Observations.ExpectedRows)
	}

	intervalBytes, intervalHash, err := readVerifiedFile(root, sourceConfig.ForecastIntervals)
	if err != nil {
		return nil, err
	}
	intervals, err := parseForecastIntervals(intervalBytes, sourceConfig.ForecastIntervals.Path, stationRows)
	if err != nil {
		return nil, err
	}

	forecastBytes, forecastHash, err := readVerifiedFile(root, sourceConfig.Forecasts.SourceFile)
	if err != nil {
		return nil, err
	}
	forecasts, err := parseForecasts(forecastBytes, sourceConfig.Forecasts.Path, intervals)
	if err != nil {
		return nil, err
	}
	if len(forecasts) != sourceConfig.Forecasts.ExpectedRows ||
		len(forecasts) != sourceConfig.ForecastIntervals.ExpectedRows {
		return nil, fmt.Errorf("forecast/interval row counts (%d/%d) do not match expected source counts (%d/%d)",
			len(forecasts), len(intervals), sourceConfig.Forecasts.ExpectedRows, sourceConfig.ForecastIntervals.ExpectedRows)
	}

	sources := map[string]SourceRecord{
		"observations": {
			Kind: ObservedSource, Path: sourceConfig.Observations.Path,
			SHA256: observationHash, Rows: len(observations),
		},
		"forecasts": {
			Kind: ForecastSource, Path: sourceConfig.Forecasts.Path,
			SHA256: forecastHash, Rows: len(forecasts),
		},
		"forecastIntervals": {
			Kind: ForecastSource, Path: sourceConfig.ForecastIntervals.Path,
			SHA256: intervalHash, Rows: len(intervals),
		},
		"stations": {
			Kind: ReferenceSource, Path: sourceConfig.Stations.Path,
			SHA256: stationHash, Rows: len(stationRows),
		},
	}

	return &Dataset{
		Stations:     orderedStations(stationRows),
		Observations: observations,
		Forecasts:    forecasts,
		RiskBands:    riskBands,
		Manifest: Manifest{
			Timezone:                 sourceConfig.Timezone,
			ForecastGeneratedAt:      nil,
			ForecastGenerationStatus: sourceConfig.Forecasts.GenerationTime,
			SelectedForecastScaling:  sourceConfig.Forecasts.SelectedScalingAlreadyApplied,
			Sources:                  sources,
			ClimateVerse:             sourceConfig.ClimateVerseCatalogue,
		},
	}, nil
}

func readVerifiedFile(root string, source config.SourceFile) ([]byte, string, error) {
	path := filepath.Join(root, filepath.FromSlash(source.Path))
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read registered source %q: %w", source.Path, err)
	}
	digest := sha256.Sum256(content)
	hash := hex.EncodeToString(digest[:])
	if hash != source.SHA256 {
		return nil, "", fmt.Errorf("registered source %q SHA-256 mismatch: got %s", source.Path, hash)
	}
	return content, hash, nil
}

func parseStations(content []byte, sourceName string) (map[string]Station, error) {
	stations := make(map[string]Station, len(expectedStations))
	err := readCSV(content, sourceName,
		[]string{"location_id", "station_name", "selected_sensor", "latitude", "longitude", "timezone"},
		func(record []string, row int) error {
			locationID := record[0]
			expected, ok := expectedStations[locationID]
			if !ok {
				return fmt.Errorf("unknown canonical station ID %q", locationID)
			}
			if _, duplicate := stations[locationID]; duplicate {
				return fmt.Errorf("duplicate station ID %q", locationID)
			}
			if record[1] != expected.name {
				return fmt.Errorf("station %q name mismatch: got %q", locationID, record[1])
			}
			if record[2] != expected.sensor {
				return fmt.Errorf("station %q sensor mismatch: got %q", locationID, record[2])
			}
			if record[5] != "Asia/Kolkata" {
				return fmt.Errorf("station %q timezone must be Asia/Kolkata", locationID)
			}
			latitude, err := parseFiniteFloat(record[3], "latitude")
			if err != nil {
				return err
			}
			longitude, err := parseFiniteFloat(record[4], "longitude")
			if err != nil {
				return err
			}
			if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
				return fmt.Errorf("station %q coordinates are out of range", locationID)
			}
			stations[locationID] = Station{
				LocationID: locationID, Name: record[1], SensorID: record[2],
				Latitude: latitude, Longitude: longitude, Timezone: record[5],
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(stations) != len(expectedStations) {
		return nil, fmt.Errorf("canonical station count %d does not match required count %d", len(stations), len(expectedStations))
	}
	return stations, nil
}

func parseObservations(content []byte, sourceName string, stations map[string]Station) ([]Observation, error) {
	observations := make([]Observation, 0, 48400)
	seen := make(map[string]struct{}, 48400)
	err := readCSV(content, sourceName,
		[]string{"timestamp_ist", "mean", "count", "available_at", "pm25", "location_id", "sensor_id", "station_name", "series_id", "flag_high_1000"},
		func(record []string, row int) error {
			station, ok := stations[record[5]]
			if !ok {
				return fmt.Errorf("observation references unknown station %q", record[5])
			}
			if record[6] != station.SensorID || record[7] != station.Name {
				return fmt.Errorf("observation station/sensor metadata mismatch for station %q", record[5])
			}
			end, err := parseObservationTimestamp(record[0])
			if err != nil {
				return fmt.Errorf("invalid observation timestamp: %w", err)
			}
			if end.Minute() != 0 || end.Second() != 0 || end.Nanosecond() != 0 {
				return fmt.Errorf("observation timestamp %q is not on an hour boundary", record[0])
			}
			if !hasISTOffset(end) {
				return fmt.Errorf("observation timestamp %q does not carry the Asia/Kolkata UTC+05:30 offset", record[0])
			}
			key := record[5] + "\x00" + end.UTC().Format(time.RFC3339Nano)
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate station-hour observation for %q at %s", record[5], end.Format(time.RFC3339))
			}
			seen[key] = struct{}{}

			mean, err := parseFiniteFloat(record[1], "mean")
			if err != nil {
				return err
			}
			value, err := parseFiniteFloat(record[4], "pm25")
			if err != nil {
				return err
			}
			if mean < 0 || value < 0 {
				return errors.New("negative PM2.5 observation is invalid")
			}
			if mean != value {
				return fmt.Errorf("mean and pm25 disagree for station %q at %s", record[5], end.Format(time.RFC3339))
			}
			countValue, err := parseFiniteFloat(record[2], "count")
			if err != nil {
				return err
			}
			if countValue <= 0 || math.Trunc(countValue) != countValue {
				return fmt.Errorf("count %q must be a positive whole number", record[2])
			}
			availableAt, err := parseObservationTimestamp(record[3])
			if err != nil {
				return fmt.Errorf("invalid available_at source timestamp: %w", err)
			}
			flagged, err := strconv.ParseBool(strings.TrimSpace(record[9]))
			if err != nil {
				return fmt.Errorf("invalid flag_high_1000 value %q: %w", record[9], err)
			}
			observations = append(observations, Observation{
				StationID: record[5], SensorID: record[6],
				IntervalStart: end.Add(-time.Hour), IntervalEnd: end,
				SourceTimestamp: end, AvailableAtSourceMTime: availableAt,
				PM25: value, Mean: mean, Count: int(countValue),
				StationName: record[7], SeriesID: record[8], FlagHigh1000: flagged,
				SourceKind: ObservedSource,
			})
			return nil
		})
	if err != nil {
		return nil, err
	}
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].StationID == observations[j].StationID {
			return observations[i].IntervalStart.Before(observations[j].IntervalStart)
		}
		return observations[i].StationID < observations[j].StationID
	})
	return observations, nil
}

type forecastInterval struct {
	id        string
	stationID string
	sensorID  string
	start     time.Time
	end       time.Time
}

func parseForecastIntervals(content []byte, sourceName string, stations map[string]Station) (map[string]forecastInterval, error) {
	intervals := make(map[string]forecastInterval, 2592)
	seenWindows := make(map[string]struct{}, 2592)
	err := readCSV(content, sourceName,
		[]string{"ID", "location_id", "sensor_id", "target_start_ist", "target_end_ist", "scoring_block", "unit"},
		func(record []string, row int) error {
			if _, duplicate := intervals[record[0]]; duplicate {
				return fmt.Errorf("duplicate Test ID %q", record[0])
			}
			station, ok := stations[record[1]]
			if !ok {
				return fmt.Errorf("Test row references unknown station %q", record[1])
			}
			if record[2] != station.SensorID {
				return fmt.Errorf("Test sensor mismatch for station %q", record[1])
			}
			if record[6] != "µg/m³" {
				return fmt.Errorf("Test row %q uses unsupported unit %q", record[0], record[6])
			}
			start, err := time.Parse(time.RFC3339Nano, record[3])
			if err != nil {
				return fmt.Errorf("invalid target_start_ist %q: %w", record[3], err)
			}
			end, err := time.Parse(time.RFC3339Nano, record[4])
			if err != nil {
				return fmt.Errorf("invalid target_end_ist %q: %w", record[4], err)
			}
			if !end.Equal(start.Add(time.Hour)) || start.Minute() != 0 || start.Second() != 0 ||
				start.Nanosecond() != 0 || end.Minute() != 0 || end.Second() != 0 || end.Nanosecond() != 0 {
				return fmt.Errorf("Test target %q is not one complete hour on the hourly grid", record[0])
			}
			if !hasISTOffset(start) || !hasISTOffset(end) {
				return fmt.Errorf("Test target %q does not use the Asia/Kolkata UTC+05:30 offset", record[0])
			}
			windowKey := record[1] + "\x00" + start.UTC().Format(time.RFC3339Nano)
			if _, duplicate := seenWindows[windowKey]; duplicate {
				return fmt.Errorf("duplicate station interval for station %q at %s", record[1], start.Format(time.RFC3339))
			}
			seenWindows[windowKey] = struct{}{}
			intervals[record[0]] = forecastInterval{
				id: record[0], stationID: record[1], sensorID: record[2], start: start, end: end,
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return intervals, nil
}

func parseForecasts(content []byte, sourceName string, intervals map[string]forecastInterval) ([]ForecastPoint, error) {
	points := make([]ForecastPoint, 0, len(intervals))
	seen := make(map[string]struct{}, len(intervals))
	err := readCSV(content, sourceName, []string{"ID", "Target"},
		func(record []string, row int) error {
			interval, ok := intervals[record[0]]
			if !ok {
				return fmt.Errorf("forecast ID %q is absent from Test.csv", record[0])
			}
			if _, duplicate := seen[record[0]]; duplicate {
				return fmt.Errorf("duplicate forecast ID %q", record[0])
			}
			value, err := parseFiniteFloat(record[1], "forecast Target")
			if err != nil {
				return err
			}
			if value < 0 {
				return fmt.Errorf("negative forecast %q is invalid", record[1])
			}
			seen[record[0]] = struct{}{}
			points = append(points, ForecastPoint{
				ID: record[0], StationID: interval.stationID, SensorID: interval.sensorID,
				IntervalStart: interval.start, IntervalEnd: interval.end,
				PM25: value, SourceKind: ForecastSource, SourceRow: row,
			})
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(seen) != len(intervals) {
		return nil, fmt.Errorf("forecast CSV has %d IDs but Test.csv has %d; missing predictions", len(seen), len(intervals))
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].StationID == points[j].StationID {
			return points[i].IntervalStart.Before(points[j].IntervalStart)
		}
		return points[i].StationID < points[j].StationID
	})
	return points, nil
}

func readCSV(content []byte, sourceName string, expectedHeader []string, consume func([]string, int) error) error {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = len(expectedHeader)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read CSV header from %q: %w", sourceName, err)
	}
	if len(header) != len(expectedHeader) {
		return fmt.Errorf("CSV header in %q has %d fields; expected %d", sourceName, len(header), len(expectedHeader))
	}
	for i, expected := range expectedHeader {
		if header[i] != expected {
			return fmt.Errorf("CSV header in %q field %d is %q; expected %q", sourceName, i+1, header[i], expected)
		}
	}

	row := 1
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		row++
		if err != nil {
			return fmt.Errorf("read CSV %q data row %d: %w", sourceName, row, err)
		}
		if err := consume(record, row); err != nil {
			return fmt.Errorf("validate CSV %q data row %d: %w", sourceName, row, err)
		}
	}
}

func parseObservationTimestamp(value string) (time.Time, error) {
	var lastError error
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05-07:00"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastError = err
	}
	return time.Time{}, lastError
}

func hasISTOffset(value time.Time) bool {
	_, offset := value.Zone()
	return offset == 5*60*60+30*60
}

func parseFiniteFloat(value, field string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q: %w", field, value, err)
	}
	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, fmt.Errorf("%s must be finite", field)
	}
	return parsed, nil
}

func orderedStations(stations map[string]Station) []Station {
	ordered := make([]Station, 0, len(stations))
	for _, id := range stationOrder {
		ordered = append(ordered, stations[id])
	}
	return ordered
}
