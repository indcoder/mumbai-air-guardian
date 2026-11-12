package beacon

import (
	"bytes"
	"math"
	"testing"
	"time"

	"mumbai-air-guardian/backend/internal/config"
)

func validPayload() Payload {
	return Payload{
		Version: ProtocolVersion, StationID: "11611", PM25: 45.25, Band: "elevated",
		Trajectory: "stable", PeakHour: time.Date(2026, 11, 12, 15, 0, 0, 0, time.FixedZone("IST", 19800)),
		RGB: RGB{Red: 154, Green: 103, Blue: 0}, Pattern: PatternSinglePulse,
	}
}

func TestProtocolRoundTripAndLineBoundaries(t *testing.T) {
	payload := validPayload()
	line, err := EncodeLine(payload)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	if line[len(line)-1] != '\n' || len(line) > MaxLineBytes {
		t.Fatalf("encoded line has invalid framing or size: %q", line)
	}
	decoded, err := DecodeLine(line)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if decoded.StationID != payload.StationID || decoded.PM25 != payload.PM25 ||
		decoded.Pattern != payload.Pattern || !decoded.PeakHour.Equal(payload.PeakHour) {
		t.Fatalf("decoded payload = %+v, want %+v", decoded, payload)
	}
}

func TestDecodeLineRejectsMalformedAndUnboundedMessages(t *testing.T) {
	valid, err := EncodeLine(validPayload())
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	tests := []struct {
		name string
		line []byte
	}{
		{name: "missing newline", line: bytes.TrimSuffix(valid, []byte{'\n'})},
		{name: "multiple lines", line: append(append([]byte(nil), valid...), valid...)},
		{name: "unknown field", line: []byte(`{"version":1,"unknown":true}` + "\n")},
		{name: "trailing JSON", line: []byte(`{} {}` + "\n")},
		{name: "oversized line", line: append(bytes.Repeat([]byte{' '}, MaxLineBytes), '\n')},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeLine(test.line); err == nil {
				t.Fatal("DecodeLine unexpectedly accepted invalid input")
			}
		})
	}
}

func TestPayloadRejectsInvalidMetricBandAndPattern(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Payload)
	}{
		{name: "negative PM2.5", mutate: func(p *Payload) { p.PM25 = -1 }},
		{name: "non-finite PM2.5", mutate: func(p *Payload) { p.PM25 = math.NaN() }},
		{name: "unsupported band", mutate: func(p *Payload) { p.Band = "official_aqi" }},
		{name: "inconsistent diagnostic", mutate: func(p *Payload) { p.Pattern = PatternDiagnostic }},
		{name: "steady-only pulse", mutate: func(p *Payload) { p.SteadyOnly = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := validPayload()
			test.mutate(&payload)
			if err := payload.Validate(); err == nil {
				t.Fatal("Validate unexpectedly accepted invalid payload")
			}
		})
	}
}

func TestNewPayloadSupportsSteadyAndDiagnosticModes(t *testing.T) {
	band := config.RiskBand{Name: "high", Color: "#B54708", BeaconPattern: "double-pulse"}
	peakHour := time.Date(2026, 11, 12, 15, 0, 0, 0, time.FixedZone("IST", 19800))
	steady, err := NewPayload("11611", 64, band, "improving", peakHour, true, false)
	if err != nil {
		t.Fatalf("create steady payload: %v", err)
	}
	if steady.Pattern != PatternSteady || steady.RGB != (RGB{Red: 181, Green: 71, Blue: 8}) {
		t.Fatalf("steady payload = %+v", steady)
	}
	diagnostic, err := NewPayload("11611", 64, band, "improving", peakHour, false, true)
	if err != nil {
		t.Fatalf("create diagnostic payload: %v", err)
	}
	if !diagnostic.Diagnostic || diagnostic.Pattern != PatternDiagnostic ||
		diagnostic.RGB != (RGB{Red: 0, Green: 80, Blue: 100}) {
		t.Fatalf("diagnostic payload = %+v", diagnostic)
	}
}

func TestSimulatorStoresValidatedPayloadWithoutExternalTransport(t *testing.T) {
	simulator := NewSimulator()
	if status := simulator.Status(); status.Status != "simulated" ||
		status.Mode != "simulation" || status.LastPayload != nil {
		t.Fatalf("initial status = %+v", status)
	}
	payload := validPayload()
	status, err := simulator.Update(payload)
	if err != nil {
		t.Fatalf("update simulation: %v", err)
	}
	if status.LastPayload == nil || *status.LastPayload != payload {
		t.Fatalf("updated status = %+v", status)
	}
	payload.PM25 = 999
	if status.LastPayload.PM25 != 45.25 {
		t.Fatal("simulator status retained a mutable caller-owned payload")
	}
}
