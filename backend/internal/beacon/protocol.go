package beacon

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"mumbai-air-guardian/backend/internal/config"
)

const (
	ProtocolVersion = 1
	MaxLineBytes    = 512
)

type Pattern string

const (
	PatternSteady      Pattern = "steady"
	PatternSinglePulse Pattern = "single-pulse"
	PatternDoublePulse Pattern = "double-pulse"
	PatternTriplePulse Pattern = "triple-pulse"
	PatternDiagnostic  Pattern = "diagnostic"
)

type RGB struct {
	Red   uint8 `json:"red"`
	Green uint8 `json:"green"`
	Blue  uint8 `json:"blue"`
}

type Payload struct {
	Version    int       `json:"version"`
	StationID  string    `json:"stationId"`
	PM25       float64   `json:"pm25"`
	Band       string    `json:"band"`
	Trajectory string    `json:"trajectory"`
	PeakHour   time.Time `json:"peakHour"`
	RGB        RGB       `json:"rgb"`
	Pattern    Pattern   `json:"pattern"`
	Diagnostic bool      `json:"diagnostic"`
	SteadyOnly bool      `json:"steadyOnly"`
}

type Status struct {
	Status      string   `json:"status"`
	Mode        string   `json:"mode"`
	Reason      string   `json:"reason"`
	LastPayload *Payload `json:"lastPayload"`
}

func NewPayload(
	stationID string,
	pm25 float64,
	band config.RiskBand,
	trajectory string,
	peakHour time.Time,
	steadyOnly bool,
	diagnostic bool,
) (Payload, error) {
	color, err := parseColor(band.Color)
	if err != nil {
		return Payload{}, err
	}
	pattern := Pattern(band.BeaconPattern)
	if diagnostic {
		pattern = PatternDiagnostic
		color = RGB{Red: 0, Green: 80, Blue: 100}
	} else if steadyOnly {
		pattern = PatternSteady
	}
	payload := Payload{
		Version: ProtocolVersion, StationID: stationID, PM25: pm25, Band: band.Name,
		Trajectory: trajectory, PeakHour: peakHour, RGB: color, Pattern: pattern,
		Diagnostic: diagnostic, SteadyOnly: steadyOnly,
	}
	if err := payload.Validate(); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func (p Payload) Validate() error {
	if p.Version != ProtocolVersion {
		return fmt.Errorf("unsupported protocol version %d", p.Version)
	}
	if strings.TrimSpace(p.StationID) == "" || len(p.StationID) > 64 {
		return errors.New("stationId must contain 1 to 64 non-whitespace characters")
	}
	if math.IsNaN(p.PM25) || math.IsInf(p.PM25, 0) || p.PM25 < 0 {
		return errors.New("pm25 must be finite and non-negative")
	}
	if p.Band != "low" && p.Band != "elevated" && p.Band != "high" && p.Band != "very_high" {
		return fmt.Errorf("unsupported risk band %q", p.Band)
	}
	if p.Trajectory != "unavailable" && p.Trajectory != "improving" &&
		p.Trajectory != "stable" && p.Trajectory != "worsening" {
		return fmt.Errorf("unsupported trajectory %q", p.Trajectory)
	}
	if p.PeakHour.IsZero() {
		return errors.New("peakHour is required")
	}
	switch p.Pattern {
	case PatternSteady, PatternSinglePulse, PatternDoublePulse, PatternTriplePulse:
		if p.Diagnostic {
			return errors.New("diagnostic payload must use the diagnostic pattern")
		}
		if p.SteadyOnly && p.Pattern != PatternSteady {
			return errors.New("steadyOnly payload must use the steady pattern")
		}
	case PatternDiagnostic:
		if !p.Diagnostic {
			return errors.New("diagnostic pattern requires diagnostic=true")
		}
	default:
		return fmt.Errorf("unsupported pattern %q", p.Pattern)
	}
	return nil
}

func EncodeLine(payload Payload) ([]byte, error) {
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode beacon payload: %w", err)
	}
	if len(encoded)+1 > MaxLineBytes {
		return nil, fmt.Errorf("beacon payload exceeds %d bytes", MaxLineBytes)
	}
	return append(encoded, '\n'), nil
}

func DecodeLine(line []byte) (Payload, error) {
	if len(line) == 0 || len(line) > MaxLineBytes || line[len(line)-1] != '\n' ||
		bytes.Count(line, []byte{'\n'}) != 1 {
		return Payload{}, fmt.Errorf("beacon message must be one newline-terminated line of at most %d bytes", MaxLineBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimSuffix(line, []byte{'\n'})))
	decoder.DisallowUnknownFields()
	var payload Payload
	if err := decoder.Decode(&payload); err != nil {
		return Payload{}, fmt.Errorf("decode beacon payload: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err != nil {
			return Payload{}, fmt.Errorf("decode trailing beacon data: %w", err)
		}
		return Payload{}, errors.New("beacon message must contain exactly one JSON object")
	}
	if err := payload.Validate(); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func parseColor(value string) (RGB, error) {
	if len(value) != 7 || value[0] != '#' {
		return RGB{}, fmt.Errorf("invalid beacon color %q", value)
	}
	decoded := make([]byte, 3)
	if _, err := hex.Decode(decoded, []byte(value[1:])); err != nil {
		return RGB{}, fmt.Errorf("invalid beacon color %q: %w", value, err)
	}
	return RGB{Red: decoded[0], Green: decoded[1], Blue: decoded[2]}, nil
}
