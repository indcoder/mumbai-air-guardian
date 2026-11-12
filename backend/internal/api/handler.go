package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"mumbai-air-guardian/backend/internal/activity"
	"mumbai-air-guardian/backend/internal/airdata"
	"mumbai-air-guardian/backend/internal/beacon"
	"mumbai-air-guardian/backend/internal/config"
	"mumbai-air-guardian/backend/internal/evidence"
	"mumbai-air-guardian/backend/internal/gemma"
	"mumbai-air-guardian/backend/internal/outlook"
)

const (
	maxRequestBytes = 16 << 10
	maxRange        = 90 * 24 * time.Hour
)

type Handler struct {
	dataset       *airdata.Dataset
	planner       *activity.Planner
	outlookConfig config.OutlookConfig
	allowedOrigin []string
	explainer     gemma.Explainer
	beacon        *beacon.Simulator
	mux           *http.ServeMux
}

func NewHandler(dataset *airdata.Dataset, outlookConfig config.OutlookConfig, allowedOrigins []string) (*Handler, error) {
	planner, err := activity.NewPlanner(dataset)
	if err != nil {
		return nil, fmt.Errorf("initialize activity planner: %w", err)
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:4200"}
	}
	for _, origin := range allowedOrigins {
		if origin == "*" || strings.TrimSpace(origin) != origin {
			return nil, errors.New("CORS origins must be explicit, non-wildcard origins")
		}
	}

	handler := &Handler{
		dataset: dataset, planner: planner, outlookConfig: outlookConfig,
		allowedOrigin: slices.Clone(allowedOrigins), mux: http.NewServeMux(),
		explainer: gemma.NewDisabledService("not_configured"), beacon: beacon.NewSimulator(),
	}
	handler.registerRoutes()
	return handler, nil
}

func (h *Handler) SetExplainer(explainer gemma.Explainer) error {
	if explainer == nil {
		return errors.New("an explanation service is required")
	}
	h.explainer = explainer
	return nil
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	writer.Header().Set("Cache-Control", "no-store")
	origin := request.Header.Get("Origin")
	if origin != "" {
		if !slices.Contains(h.allowedOrigin, origin) {
			writeError(writer, http.StatusForbidden, "origin_not_allowed", "Request origin is not allowed")
			return
		}
		writer.Header().Set("Access-Control-Allow-Origin", origin)
		writer.Header().Set("Vary", "Origin")
	}
	if request.Method == http.MethodOptions {
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	h.mux.ServeHTTP(writer, request)
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("GET /api/health", h.health)
	h.mux.HandleFunc("GET /api/stations", h.stations)
	h.mux.HandleFunc("GET /api/stations/{id}/observations", h.observations)
	h.mux.HandleFunc("GET /api/stations/{id}/forecast", h.forecast)
	h.mux.HandleFunc("GET /api/stations/{id}/air-outlook", h.airOutlook)
	h.mux.HandleFunc("GET /api/stations/{id}/context", h.context)
	h.mux.HandleFunc("POST /api/activity-plan", h.activityPlan)
	h.mux.HandleFunc("POST /api/explain", h.explain)
	h.mux.HandleFunc("GET /api/sources", h.sources)
	h.mux.HandleFunc("GET /api/ai/status", h.aiStatus)
	h.mux.HandleFunc("GET /api/beacon/status", h.beaconStatus)
	h.mux.HandleFunc("POST /api/beacon", h.beaconAction)
}

func (h *Handler) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"status": "ok",
		"capabilities": map[string]any{
			"observations": map[string]string{"status": "available"},
			"forecasts":    map[string]string{"status": "available", "generationTime": "unavailable"},
			"weather":      map[string]string{"status": "unavailable", "reason": "not validated"},
			"ai": map[string]string{
				"status": "available",
				"reason": "Deterministic fallback is available; local Gemma status is reported separately",
			},
			"beacon": map[string]string{
				"status": "simulated", "reason": "Virtual output only; no physical serial device is opened.",
			},
			"activityPlanning": map[string]string{
				"status": "available", "newPlansRequire": "local Go service",
			},
		},
	})
}

func (h *Handler) stations(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"stations": h.dataset.Stations})
}

func (h *Handler) observations(writer http.ResponseWriter, request *http.Request) {
	stationID := request.PathValue("id")
	if !h.hasStation(stationID) {
		writeError(writer, http.StatusNotFound, "unknown_station", "Station was not found")
		return
	}
	from, to, ok := parseTimeRange(writer, request)
	if !ok {
		return
	}
	points := make([]airdata.Observation, 0)
	excluded := 0
	for _, observation := range h.dataset.Observations {
		if observation.StationID != stationID ||
			observation.IntervalStart.Before(from) || observation.IntervalEnd.After(to) {
			continue
		}
		points = append(points, observation)
		if !observation.SummaryEligible() {
			excluded++
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"stationId": stationID, "from": from, "to": to, "points": points,
		"excludedFlaggedCount": excluded,
		"exclusionReason":      "Explicit source quality flag flag_high_1000=true",
		"source":               h.dataset.Manifest.Sources["observations"],
	})
}

func (h *Handler) forecast(writer http.ResponseWriter, request *http.Request) {
	stationID := request.PathValue("id")
	if !h.hasStation(stationID) {
		writeError(writer, http.StatusNotFound, "unknown_station", "Station was not found")
		return
	}
	from, to, ok := parseTimeRange(writer, request)
	if !ok {
		return
	}
	points := make([]airdata.ForecastPoint, 0)
	for _, point := range h.dataset.Forecasts {
		if point.StationID == stationID &&
			!point.IntervalStart.Before(from) && !point.IntervalEnd.After(to) {
			points = append(points, point)
		}
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"stationId": stationID, "from": from, "to": to, "points": points,
		"generatedAt":      h.dataset.Manifest.ForecastGeneratedAt,
		"generationStatus": h.dataset.Manifest.ForecastGenerationStatus,
		"source":           h.dataset.Manifest.Sources["forecasts"],
	})
}

func (h *Handler) airOutlook(writer http.ResponseWriter, request *http.Request) {
	stationID := request.PathValue("id")
	if !h.hasStation(stationID) {
		writeError(writer, http.StatusNotFound, "unknown_station", "Station was not found")
		return
	}
	result, err := outlook.BuildAirOutlook(
		stationID, h.dataset.Observations, h.dataset.Forecasts,
		h.dataset.RiskBands, h.outlookConfig.Trajectory.LookaheadHours,
		h.outlookConfig.Trajectory.SlopeThresholdUGM3PerHour,
	)
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "outlook_unavailable", "A complete station outlook is unavailable")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (h *Handler) activityPlan(writer http.ResponseWriter, request *http.Request) {
	body, ok := decodeActivityRequest(writer, request)
	if !ok {
		return
	}
	result, err := h.planner.Plan(body)
	if err != nil {
		writeActivityError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (h *Handler) explain(writer http.ResponseWriter, request *http.Request) {
	language := request.URL.Query().Get("lang")
	if language != gemma.English && language != gemma.Marathi && language != gemma.Hindi {
		writeError(writer, http.StatusBadRequest, "invalid_language", "lang must be en, mr, or hi")
		return
	}
	body, ok := decodeActivityRequest(writer, request)
	if !ok {
		return
	}
	plan, err := h.planner.Plan(body)
	if err != nil {
		writeActivityError(writer, err)
		return
	}
	activityEvidence, err := evidence.BuildActivity(h.dataset, body, plan, language)
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "explanation_unavailable", "Verified explanation evidence is unavailable")
		return
	}
	result, err := h.explainer.Explain(request.Context(), activityEvidence, language)
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "explanation_unavailable", "The explanation service could not complete this request")
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func decodeActivityRequest(writer http.ResponseWriter, request *http.Request) (activity.Request, bool) {
	if !isJSONContentType(request.Header.Get("Content-Type")) {
		writeError(writer, http.StatusUnsupportedMediaType, "content_type_required", "Content-Type must be application/json")
		return activity.Request{}, false
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var body activity.Request
	if err := decoder.Decode(&body); err != nil {
		writeActivityError(writer, err)
		return activity.Request{}, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			writeActivityError(writer, err)
			return activity.Request{}, false
		}
		writeError(writer, http.StatusBadRequest, "invalid_json", "Request body must contain exactly one JSON object")
		return activity.Request{}, false
	}
	return body, true
}

func (h *Handler) sources(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"snapshotLabel":          "Challenge forecast snapshot",
		"latestObservationLabel": "Latest observation in supplied dataset",
		"forecastGenerationTime": "Forecast generation time unavailable",
		"manifest":               h.dataset.Manifest,
	})
}

func (h *Handler) aiStatus(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, h.explainer.Status(request.Context()))
}

func (h *Handler) beaconStatus(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, h.beacon.Status())
}

type beaconActionRequest struct {
	StationID  string `json:"stationId"`
	SteadyOnly bool   `json:"steadyOnly"`
	Diagnostic bool   `json:"diagnostic"`
}

func (h *Handler) beaconAction(writer http.ResponseWriter, request *http.Request) {
	body, ok := decodeBeaconAction(writer, request)
	if !ok {
		return
	}
	if !h.hasStation(body.StationID) {
		writeError(writer, http.StatusNotFound, "unknown_station", "Station was not found")
		return
	}
	airOutlook, err := outlook.BuildAirOutlook(
		body.StationID, h.dataset.Observations, h.dataset.Forecasts,
		h.dataset.RiskBands, h.outlookConfig.Trajectory.LookaheadHours,
		h.outlookConfig.Trajectory.SlopeThresholdUGM3PerHour,
	)
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "beacon_unavailable", "A verified forecast pattern is unavailable")
		return
	}
	var peakHour time.Time
	for _, point := range h.dataset.Forecasts {
		if point.StationID == body.StationID &&
			!point.IntervalStart.Before(airOutlook.LatestObservation.IntervalEnd) &&
			point.PM25 == airOutlook.Peak.ValuePM25 &&
			(peakHour.IsZero() || point.IntervalStart.Before(peakHour)) {
			peakHour = point.IntervalStart
		}
	}
	if peakHour.IsZero() {
		writeError(writer, http.StatusServiceUnavailable, "beacon_unavailable", "The forecast peak interval is unavailable")
		return
	}
	payload, err := beacon.NewPayload(
		body.StationID, airOutlook.Peak.ValuePM25, airOutlook.Peak.Band,
		string(airOutlook.Trajectory.Status), peakHour, body.SteadyOnly, body.Diagnostic,
	)
	if err != nil {
		writeError(writer, http.StatusServiceUnavailable, "beacon_unavailable", "The configured beacon pattern is invalid")
		return
	}
	status, err := h.beacon.Update(payload)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_beacon_payload", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, status)
}

func decodeBeaconAction(writer http.ResponseWriter, request *http.Request) (beaconActionRequest, bool) {
	if !isJSONContentType(request.Header.Get("Content-Type")) {
		writeError(writer, http.StatusUnsupportedMediaType, "content_type_required", "Content-Type must be application/json")
		return beaconActionRequest{}, false
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var body beaconActionRequest
	if err := decoder.Decode(&body); err != nil {
		status := http.StatusBadRequest
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(writer, status, "invalid_beacon_request", "Request must be one valid beacon action object")
		return beaconActionRequest{}, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeError(writer, http.StatusBadRequest, "invalid_beacon_request", "Request body must contain exactly one JSON object")
		return beaconActionRequest{}, false
	}
	if strings.TrimSpace(body.StationID) == "" {
		writeError(writer, http.StatusBadRequest, "invalid_beacon_request", "stationId is required")
		return beaconActionRequest{}, false
	}
	return body, true
}

func (h *Handler) context(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"status": "unavailable",
		"reason": "Environmental context is disabled pending mapping, units, timing, and coverage validation",
	})
}

func (h *Handler) hasStation(id string) bool {
	for _, station := range h.dataset.Stations {
		if station.LocationID == id {
			return true
		}
	}
	return false
}

func parseTimeRange(writer http.ResponseWriter, request *http.Request) (time.Time, time.Time, bool) {
	from, err := time.Parse(time.RFC3339Nano, request.URL.Query().Get("from"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_from", "from must be an RFC3339 IST timestamp")
		return time.Time{}, time.Time{}, false
	}
	to, err := time.Parse(time.RFC3339Nano, request.URL.Query().Get("to"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_to", "to must be an RFC3339 IST timestamp")
		return time.Time{}, time.Time{}, false
	}
	if !validISTHour(from) || !validISTHour(to) || !from.Before(to) || to.Sub(from) > maxRange {
		writeError(writer, http.StatusBadRequest, "invalid_range", "from and to must be ordered IST hour boundaries within a 90-day range")
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

func validISTHour(value time.Time) bool {
	_, offset := value.Zone()
	return offset == 5*60*60+30*60 &&
		value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0
}

func isJSONContentType(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.EqualFold(strings.TrimSpace(mediaType), "application/json")
}

func writeActivityError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, activity.ErrUnknownStation):
		writeError(writer, http.StatusNotFound, "unknown_station", "Station was not found")
	case errors.Is(err, activity.ErrInvalidRequest):
		writeError(writer, http.StatusUnprocessableEntity, "invalid_activity_request", err.Error())
	case errors.Is(err, activity.ErrInsufficientCoverage):
		writeError(writer, http.StatusUnprocessableEntity, "insufficient_coverage", err.Error())
	case isRequestTooLarge(err):
		writeError(writer, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds 16 KiB")
	default:
		writeError(writer, http.StatusBadRequest, "invalid_json", "Request body is invalid")
	}
}

func isRequestTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]string{"code": code, "message": message})
}
