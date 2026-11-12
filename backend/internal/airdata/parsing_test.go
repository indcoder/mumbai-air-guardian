package airdata

import (
	"strings"
	"testing"
	"time"
)

var testStations = map[string]Station{
	"11611": {
		LocationID: "11611",
		Name:       "Malad West, Mumbai - IITM",
		SensorID:   "12236426",
		Timezone:   "Asia/Kolkata",
	},
}

func observationCSV(rows ...string) []byte {
	return []byte("timestamp_ist,mean,count,available_at,pm25,location_id,sensor_id,station_name,series_id,flag_high_1000\n" +
		strings.Join(rows, "\n") + "\n")
}

func TestObservationAdapterRejectsInvalidOrMixedSources(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "negative PM2.5",
			data: observationCSV("2026-11-10 09:00:00+05:30,-1,4,2026-10-20T00:00:37+05:30,-1,11611,12236426,\"Malad West, Mumbai - IITM\",openaq:12236426,False"),
		},
		{
			name: "duplicate station hour",
			data: observationCSV(
				"2026-11-10 09:00:00+05:30,101,4,2026-10-20T00:00:37+05:30,101,11611,12236426,\"Malad West, Mumbai - IITM\",openaq:12236426,False",
				"2026-11-10 09:00:00+05:30,102,4,2026-10-20T00:00:37+05:30,102,11611,12236426,\"Malad West, Mumbai - IITM\",openaq:12236426,False",
			),
		},
		{
			name: "submission is not an observation source",
			data: []byte("ID,Target\nMUM_11611_20261110T1000_IST,74.129998\n"),
		},
		{
			name: "malformed quality flag",
			data: observationCSV("2026-11-10 09:00:00+05:30,101,4,2026-10-20T00:00:37+05:30,101,11611,12236426,\"Malad West, Mumbai - IITM\",openaq:12236426,maybe"),
		},
		{
			name: "non-IST offset",
			data: observationCSV("2026-11-10 09:00:00+00:00,101,4,2026-10-20T00:00:37+05:30,101,11611,12236426,\"Malad West, Mumbai - IITM\",openaq:12236426,False"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseObservations(test.data, "fixture.csv", testStations); err == nil {
				t.Fatal("invalid observation input was accepted")
			}
		})
	}
}

func TestObservationAdapterRetainsUnflaggedValuesAboveFiveHundred(t *testing.T) {
	data := observationCSV("2026-11-10 09:00:00+05:30,501,4,2026-10-20T00:00:37+05:30,501,11611,12236426,\"Malad West, Mumbai - IITM\",openaq:12236426,False")
	observations, err := parseObservations(data, "fixture.csv", testStations)
	if err != nil {
		t.Fatalf("parse unflagged value above 500: %v", err)
	}
	if len(observations) != 1 || observations[0].PM25 != 501 || observations[0].FlagHigh1000 {
		t.Fatalf("unflagged source value was changed or removed: %+v", observations)
	}
}

func TestForecastAdapterRejectsDuplicateAndMissingIDJoins(t *testing.T) {
	interval := forecastInterval{
		id: "MUM_11611_20261110T1000_IST", stationID: "11611", sensorID: "12236426",
		start: parseFixtureTime(t, "2026-11-10T09:00:00+05:30"),
		end:   parseFixtureTime(t, "2026-11-10T10:00:00+05:30"),
	}
	intervals := map[string]forecastInterval{interval.id: interval}
	duplicate := []byte("ID,Target\n" + interval.id + ",74.129998\n" + interval.id + ",74.129998\n")
	if _, err := parseForecasts(duplicate, "fixture.csv", intervals); err == nil {
		t.Fatal("duplicate forecast ID was accepted")
	}
	if _, err := parseForecasts([]byte("ID,Target\n"), "fixture.csv", intervals); err == nil {
		t.Fatal("missing forecast prediction was accepted")
	}
	if _, err := parseForecasts([]byte("ID,Target\n"+interval.id+",-1\n"), "fixture.csv", intervals); err == nil {
		t.Fatal("negative forecast was accepted")
	}
}

func parseFixtureTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
