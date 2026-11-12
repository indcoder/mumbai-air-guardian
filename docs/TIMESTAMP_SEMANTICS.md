# Timestamp and interval semantics

The application has four registered runtime inputs:

| Input | Runtime meaning |
| --- | --- |
| `zindi_artifacts\Train.csv` | Measured hourly observations; `timestamp_ist` is the interval end. |
| `zindi_artifacts\submission_xgb_corrected_scale_110.csv` | Forecast values only. The selected 1.10 scaling is already applied. |
| `zindi_artifacts\Test.csv` | Authoritative forecast ID-to-station and half-open interval mapping. |
| `zindi_artifacts\Stations.csv` | Canonical station, sensor, coordinates and timezone mapping. |

Other submissions, validation predictions, backtests, overlap exports and
weather files are not observation inputs.

## Observations

For each `Train.csv` row:

```text
intervalEnd   = timestamp_ist
intervalStart = timestamp_ist - 1 hour
interval      = [intervalStart, intervalEnd)
```

For example, a source timestamp of `2026-11-10 09:00:00+05:30` describes
the measured interval **08:00-09:00 IST**. Retain the supplied measurement
and quality flag; do not fill its missing neighboring hours.

`available_at` is retained source-file modification metadata. It is not a
verified first-publication time or forecast-issuance time.

## Forecasts

Read each forecast value from the selected submission file and join it by
`ID` to `Test.csv`. Use `target_start_ist` and `target_end_ist` directly;
do not derive a forecast start by parsing the timestamp embedded in the ID.

`MUM_11611_20261027T0100_IST` ends at 01:00 IST and maps through Test to
**27 October 2026, 00:00-01:00 IST**, not 01:00-02:00.

The model's generation/issuance time is unavailable. Keep `generatedAt`
nullable and display **Forecast generation time unavailable**. Do not infer
it from file modification, import, startup or cache timestamps.

## Observed/forecast boundary

For Malad West (`11611`):

- Final supplied observed interval: **10 November 2026, 08:00-09:00 IST**.
- First subsequent forecast interval: **10 November 2026, 09:00-10:00 IST**.
- Shared chart boundary: **10 November 2026, 09:00 IST**.

Render observations as a solid series and forecasts as a dashed series with
interval-aware tooltips and a marked 09:00 boundary. Do not connect the
observed and forecast series. Missing intervals remain visible gaps,
including the challenge forecast gap from 24 through 26 October and all
observed-data gaps.

## Validation and preservation

`config\sources.yaml` pins the SHA-256 digest and expected row count for each
of the four approved input files. Startup rejects changed input files,
unexpected schemas, station/sensor mismatches, duplicate station-hour
records, duplicate forecast intervals or IDs, incomplete ID joins,
unsupported units, negative values and invalid timestamps. All existing
files in `zindi_artifacts` remain read-only application inputs.
