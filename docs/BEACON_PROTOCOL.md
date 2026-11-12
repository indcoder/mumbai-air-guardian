# Air Beacon simulation and protocol boundary

## Current capability

Air Beacon is a **simulation only**. The Go service stores the last validated virtual payload in memory. It does not open, scan, or communicate with a serial port, and no Arduino firmware or physical-device transport is implemented. The preview is informational and is not a sensor, alert service, or medical recommendation.

## Forecast action

The browser sends an explicit `POST /api/beacon` request:

```json
{"stationId":"11611","steadyOnly":false,"diagnostic":false}
```

Only `stationId`, `steadyOnly`, and `diagnostic` are accepted. Unknown properties are rejected. The server verifies the station and derives PM2.5, risk band, trajectory, peak interval and output color from the loaded, validated dataset. Clients cannot submit or override those values. `steadyOnly` selects a non-pulsing pattern for a forecast preview. `diagnostic` selects a diagnostic-only visual pattern and takes precedence over the steady-only option; it does not read or change air-quality data.

`GET /api/beacon/status` returns `status: "simulated"` and `mode: "simulation"`, plus the last payload or `null`. The service's ordinary health remains independent of this optional capability.

## Version 1 payload

The Go codec serializes exactly one UTF-8 JSON object as a newline-terminated line, at most 512 bytes including the newline. It rejects missing or multiple lines, trailing JSON, unknown properties, invalid risk bands/patterns, non-finite or negative PM2.5, missing peak time, and inconsistent diagnostic/steady flags.

| Field | Meaning |
| --- | --- |
| `version` | Protocol version, currently `1`. |
| `stationId` | Selected source station identifier. |
| `pm25` | Peak PM2.5 value from the selected station's validated forecast. |
| `band` | Configured band: `low`, `elevated`, `high`, or `very_high`. |
| `trajectory` | `improving`, `stable`, `worsening`, or `unavailable`. |
| `peakHour` | RFC 3339 timestamp of the forecast interval start for the peak. |
| `rgb` | Output color derived from the configured risk band; diagnostic mode uses a fixed teal. |
| `pattern` | `steady`, `single-pulse`, `double-pulse`, `triple-pulse`, or `diagnostic`. |
| `diagnostic` | Whether the preview is a diagnostic action. |
| `steadyOnly` | Whether the preview was explicitly constrained to steady output. |

`backend/internal/beacon` owns payload validation and line encoding/decoding. Its simulator owns only an in-memory copy of the most recent validated payload. The browser renders a static, decorative color marker and localized text for the selected pattern; it does not introduce automatic or persistent motion.

## Firmware and transport status

No firmware, serial transport, device acknowledgment, or `ACK`/`ERR` exchange has been implemented. These are future, optional work and must not be inferred from the simulation protocol. Any physical integration must separately define and test bounded line reads, version/field validation, acknowledgment/error framing, non-blocking light patterns, disconnect behavior and cancellable cleanup before a device-control claim is made. Do not auto-scan ports or require hardware for application startup or tests.
