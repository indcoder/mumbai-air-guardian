package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"mumbai-air-guardian/backend/internal/activity"
	"mumbai-air-guardian/backend/internal/airdata"
	"mumbai-air-guardian/backend/internal/beacon"
	"mumbai-air-guardian/backend/internal/config"
	"mumbai-air-guardian/backend/internal/outlook"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate API test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	dataset, err := airdata.LoadFromRoot(root)
	if err != nil {
		t.Fatalf("load verified dataset: %v", err)
	}
	outlookConfig, err := config.LoadOutlookConfig(filepath.Join(root, filepath.FromSlash(config.OutlookConfigRelativePath)))
	if err != nil {
		t.Fatalf("load outlook configuration: %v", err)
	}
	handler, err := NewHandler(dataset, outlookConfig, []string{"http://localhost:4200"})
	if err != nil {
		t.Fatalf("create API handler: %v", err)
	}
	return handler
}

func TestHealthReportsLivenessSeparatelyFromOptionalCapabilities(t *testing.T) {
	handler := newTestHandler(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		Status       string `json:"status"`
		Capabilities map[string]struct {
			Status string `json:"status"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("health status = %q, want ok", body.Status)
	}
	if body.Capabilities["weather"].Status != "unavailable" ||
		body.Capabilities["ai"].Status != "available" ||
		body.Capabilities["beacon"].Status != "simulated" {
		t.Fatalf("optional capability statuses are not explicit: %+v", body.Capabilities)
	}
}

func TestBeaconSimulationUsesVerifiedForecastAndSteadyDiagnosticModes(t *testing.T) {
	handler := newTestHandler(t)
	expectedOutlook, err := outlook.BuildAirOutlook(
		"11611", handler.dataset.Observations, handler.dataset.Forecasts,
		handler.dataset.RiskBands, handler.outlookConfig.Trajectory.LookaheadHours,
		handler.outlookConfig.Trajectory.SlopeThresholdUGM3PerHour,
	)
	if err != nil {
		t.Fatalf("build expected outlook from verified forecast: %v", err)
	}
	var expectedPeakHour time.Time
	for _, point := range handler.dataset.Forecasts {
		if point.StationID == "11611" &&
			!point.IntervalStart.Before(expectedOutlook.LatestObservation.IntervalEnd) &&
			point.PM25 == expectedOutlook.Peak.ValuePM25 &&
			(expectedPeakHour.IsZero() || point.IntervalStart.Before(expectedPeakHour)) {
			expectedPeakHour = point.IntervalStart
		}
	}
	if expectedPeakHour.IsZero() {
		t.Fatal("expected forecast peak interval")
	}

	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, httptest.NewRequest(http.MethodGet, "/api/beacon/status", nil))
	var status beacon.Status
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode initial beacon status: %v", err)
	}
	if status.Status != "simulated" || status.Mode != "simulation" || status.LastPayload != nil {
		t.Fatalf("initial beacon status = %+v", status)
	}

	tests := []struct {
		name        string
		body        string
		wantPattern beacon.Pattern
		wantTest    bool
	}{
		{
			name:        "steady-only forecast",
			body:        `{"stationId":"11611","steadyOnly":true}`,
			wantPattern: beacon.PatternSteady,
		},
		{
			name:        "diagnostic test",
			body:        `{"stationId":"11611","diagnostic":true}`,
			wantPattern: beacon.PatternDiagnostic,
			wantTest:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/beacon", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("beacon action status = %d, want %d: %s", response.Code, http.StatusOK, response.Body)
			}
			var result beacon.Status
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode beacon action: %v", err)
			}
			if result.Status != "simulated" || result.Mode != "simulation" ||
				result.LastPayload == nil || result.LastPayload.Pattern != test.wantPattern ||
				result.LastPayload.Diagnostic != test.wantTest ||
				result.LastPayload.PM25 != expectedOutlook.Peak.ValuePM25 ||
				result.LastPayload.Band != expectedOutlook.Peak.Band.Name ||
				result.LastPayload.Trajectory != string(expectedOutlook.Trajectory.Status) ||
				!result.LastPayload.PeakHour.Equal(expectedPeakHour) {
				t.Fatalf("beacon action result = %+v; expected forecast peak %+v at %s", result, expectedOutlook.Peak, expectedPeakHour)
			}
		})
	}
}

func TestBeaconActionRejectsInvalidRequests(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name       string
		body       string
		content    string
		wantStatus int
	}{
		{name: "unknown field", body: `{"stationId":"11611","port":"COM1"}`, content: "application/json", wantStatus: http.StatusBadRequest},
		{name: "missing station", body: `{}`, content: "application/json", wantStatus: http.StatusBadRequest},
		{name: "unknown station", body: `{"stationId":"999"}`, content: "application/json", wantStatus: http.StatusNotFound},
		{name: "wrong content type", body: `{"stationId":"11611"}`, content: "text/plain", wantStatus: http.StatusUnsupportedMediaType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/beacon", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.content)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("beacon action status = %d, want %d: %s", response.Code, test.wantStatus, response.Body)
			}
		})
	}
}

func TestActivityPlanUsesVerifiedForecastAndOmitsAbsentComparison(t *testing.T) {
	handler := newTestHandler(t)
	body := `{"stationId":"11611","activityType":"outdoor exercise","date":"2026-11-12","earliestStart":"08:00","latestEnd":"20:00","durationMinutes":120}`
	request := httptest.NewRequest(http.MethodPost, "/api/activity-plan", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("activity-plan status = %d, want %d: %s", response.Code, http.StatusOK, response.Body)
	}
	var result activity.Result
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode activity plan: %v", err)
	}
	if result.RecommendedPeriod.Start.Format("15:04") != "15:00" ||
		result.RecommendedPeriod.End.Format("15:04") != "17:00" {
		t.Fatalf("recommended period = %s-%s, want 15:00-17:00",
			result.RecommendedPeriod.Start.Format("15:04"), result.RecommendedPeriod.End.Format("15:04"))
	}
	if result.Comparison != nil {
		t.Fatalf("comparison = %+v, want omitted when no preferred time was requested", result.Comparison)
	}
	if result.SourceForecastSHA256 == "" || result.ForecastGeneration != "unavailable" {
		t.Fatalf("forecast provenance is incomplete: %+v", result)
	}
}

func TestActivityPlanRejectsUnsupportedContentTypeAndUnknownFields(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
	}{
		{name: "unsupported content type with parameters", contentType: "text/plain; charset=utf-8", body: `{}`, wantStatus: http.StatusUnsupportedMediaType},
		{name: "unknown request field", contentType: "application/json", body: `{"unexpected":true}`, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/activity-plan", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body)
			}
		})
	}
}

func TestActivityPlanEnforcesRequestSizeLimit(t *testing.T) {
	handler := newTestHandler(t)
	body := `{"activityType":"` + strings.Repeat("a", maxRequestBytes) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/activity-plan", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusRequestEntityTooLarge, response.Body)
	}
}

func TestReadRoutesExposeDataAndExplicitUnavailableCapabilities(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name string
		path string
	}{
		{name: "stations", path: "/api/stations"},
		{name: "observations", path: "/api/stations/11611/observations?from=2026-11-10T08%3A00%3A00%2B05%3A30&to=2026-11-10T10%3A00%3A00%2B05%3A30"},
		{name: "forecast", path: "/api/stations/11611/forecast?from=2026-11-10T09%3A00%3A00%2B05%3A30&to=2026-11-10T10%3A00%3A00%2B05%3A30"},
		{name: "air outlook", path: "/api/stations/11611/air-outlook"},
		{name: "environmental context", path: "/api/stations/11611/context"},
		{name: "sources", path: "/api/sources"},
		{name: "AI status", path: "/api/ai/status"},
		{name: "beacon status", path: "/api/beacon/status"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body)
			}
			if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("Content-Type = %q, want JSON", response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestTimeRangesRequireISTHourBoundariesAndRemainBounded(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "non-IST offset", from: "2026-11-10T09:00:00Z", to: "2026-11-10T10:00:00Z"},
		{name: "sub-hour boundary", from: "2026-11-10T09:30:00+05:30", to: "2026-11-10T10:30:00+05:30"},
		{name: "range exceeds limit", from: "2026-08-01T00:00:00+05:30", to: "2026-11-10T00:00:00+05:30"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/stations/11611/observations?from=%s&to=%s", test.from, test.to)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body)
			}
		})
	}
}

func TestCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name       string
		origin     string
		wantStatus int
		wantHeader string
	}{
		{name: "configured", origin: "http://localhost:4200", wantStatus: http.StatusOK, wantHeader: "http://localhost:4200"},
		{name: "unconfigured", origin: "http://127.0.0.1:4200", wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != test.wantHeader {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, test.wantHeader)
			}
		})
	}
}
