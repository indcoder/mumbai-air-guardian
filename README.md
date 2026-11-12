# Mumbai Air Guardian

Air Pulse is a local, fixed-snapshot air-quality explorer for four Mumbai monitoring stations. The Go service reads and validates the supplied observation and forecast files; the Angular interface displays their provenance and provides a Go-backed activity-window planner.

This is not live monitoring or medical advice. The latest supplied observation ends at **10 November 2026, 09:00 IST**. Forecast issuance time is unavailable. The selected forecast file has already been scaled by 1.10; the application does not scale it again.

## Current capability status

| Capability | Status |
| --- | --- |
| Four-station observations and forecast snapshot | IMPLEMENTED |
| Hourly outlook, trend, chart and interval-table alternative | IMPLEMENTED |
| Activity planning from complete contiguous hourly forecast coverage | IMPLEMENTED |
| Preferred-time comparison | IMPLEMENTED, optional |
| New activity calculations when the Go service is offline | UNAVAILABLE |
| Live air monitoring and forecast issuance timestamp | UNAVAILABLE |
| User-triggered English, Marathi and Hindi explanations with localized fallbacks | IMPLEMENTED; Gemma generation remains English-only |
| Validated output from local Gemma | OPTIONAL; English sample verified locally; timeout or invalid output uses a template |
| English/Marathi/Hindi interface localization | IMPLEMENTED; wording accepted by the project owner |
| Offline snapshot storage and exact saved-plan replay | IMPLEMENTED; production browser snapshot restore and exact replay verified |
| Plan sharing and copy controls | IMPLEMENTED; native share sheet and clipboard copy verified manually on Windows |
| Air Beacon preview | SIMULATED; no physical device control or serial transport |

When no preferred start time is supplied, the planner returns the lowest-forecast valid period and omits the comparison. The browser does not implement a second planner.

## Prerequisites

- Go **1.27.4** for the backend.
- Node.js **22.22.3 or later in the 22.x line** and npm **10.9.9** for Angular 22.
- The original competition input files, unchanged, in a `zindi_artifacts\` folder at the project root.

### Input data is not included

The competition data files are **not redistributed in this repository** because redistribution rights have not been confirmed. Obtain them from the Mumbai air-quality competition on Zindi and place these files, unchanged, in `zindi_artifacts\`:

- `Train.csv`, `Test.csv`, `Stations.csv` — competition data.
- `submission_xgb_corrected_scale_110.csv` — the author's selected forecast output (already scaled by 1.10).

The API refuses to start if any file is missing or its SHA-256 digest or row count differs from `config\sources.yaml`. Screenshots in [`docs\submission-assets`](docs/submission-assets) show the running application.

## Run locally on Windows

Open two PowerShell windows from the project root.

In the first window, start the API:

```powershell
$go = Join-Path $env:LOCALAPPDATA 'Programs\Go\go1.27.4\go\bin\go.exe'
Push-Location .\backend
$env:AIR_GUARDIAN_DATA_ROOT = (Resolve-Path ..).Path
& $go run .\cmd\air-guardian
```

The API listens on `127.0.0.1:8080`. Startup validates the configured input paths, row counts, schemas and SHA-256 digests. Leave this window running.

For English Gemma explanations, run the existing local Ollama service with `gemma4:e4b` installed. The API defaults to `http://localhost:11434` and a 30-second limit; if generation fails, it returns a labeled template instead. Ollama is not required for activity planning.

In the second window, install frontend dependencies once and start Angular:

```powershell
$node = Join-Path $env:LOCALAPPDATA 'Programs\node-v22.23.5\node.exe'
$npmCli = Join-Path (Split-Path $node) 'node_modules\npm\bin\npm-cli.js'
Push-Location .\frontend
& $node $npmCli ci
& $node .\node_modules\@angular\cli\bin\ng.js serve
```

Open `http://localhost:4200/`. The frontend runtime configuration points to the local API at `http://localhost:8080`. Use a supported Node version before running `npm ci`; Angular rejects unsupported Node versions.

Stop either development server with `Ctrl+C`.

## Verify

From `backend` (with the `$go` variable from the API command):

```powershell
& $go test -count=1 ./...
& $go vet ./...
& $go build ./...
```

From `frontend` (with the `$node` variable from the Angular command):

```powershell
& $node .\node_modules\@angular\cli\bin\ng.js build
& $node .\node_modules\@angular\cli\bin\ng.js test --watch=false
```

The variables above use the verified user-local Go 1.27.4 and Node 22.23.5 paths in this environment. If installed elsewhere, update them to point to a compatible Go executable and Node runtime.

## Data and design notes

- `config\sources.yaml` names the observation, Test interval-reference, selected forecast and station files and pins their expected hashes and row counts.
- Observation timestamps and timestamps embedded in forecast IDs denote interval ends. Forecasts are joined to `Test.csv` by ID; see [`docs\TIMESTAMP_SEMANTICS.md`](docs/TIMESTAMP_SEMANTICS.md).
- Source-flagged observations remain visible for diagnosis but are excluded from ordinary summaries.
- All source dates and times use `Asia/Kolkata`. Missing intervals remain gaps; the planner requires complete contiguous hourly coverage.
- Beacon output is a virtual preview only. It derives its pattern and peak interval from the selected station's verified forecast; no physical device, serial transport, or Arduino firmware is connected. See [`docs\BEACON_PROTOCOL.md`](docs/BEACON_PROTOCOL.md) for the versioned payload contract and implementation boundary. Weather context remains unavailable.
- Explanations are requested only after selecting **Explain this plan**; English, Marathi and Hindi templates are available, while Gemma generation is English-only. A local English response has passed the application's checks and been reviewed; those checks cannot guarantee every model claim is correct.
- The interface is built with accessibility in mind, but this implementation has not received a full professional accessibility audit. See [`docs\MODERN_WEB_REVIEW.md`](docs/MODERN_WEB_REVIEW.md) for completed checks and remaining manual testing.

## Project map

- `backend\` — Go ingestion, outlook, planner and HTTP API.
- `frontend\` — Angular application and tests.
- `config\` — pinned source and outlook configuration.
- `docs\` — timestamp, beacon protocol boundary and focused web-accessibility review.
- `zindi_artifacts\` — competition inputs (not committed; see [Input data is not included](#input-data-is-not-included)).

The app and demo are local-development artifacts; production hosting and live-data integrations are not included.
