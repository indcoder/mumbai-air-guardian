# Mumbai Air Guardian

## Title and tagline

**Mumbai Air Guardian: Air Pulse**  
*A local, evidence-based activity planner for a fixed Mumbai air-quality forecast.*

## Project summary

Air Pulse is a local prototype for exploring the supplied air-quality observations and forecast snapshot for four Mumbai monitoring stations. The Angular interface presents interval-aware data and asks the Go service to compare complete forecast windows for a planned activity. It is not a live monitoring service or a source of medical advice.

## Problem

Air-quality datasets can be misread when observation intervals, forecast intervals, missing periods, scaling, and forecast issuance time are unclear. Air Pulse makes those boundaries visible and provides a consistent way to compare forecast periods without describing archived inputs as current conditions.

## Impact

**Potential impact; not measured:** the prototype is designed to help people inspect supplied forecasts and compare possible activity times. No health outcome, behavior change, deployment reach, or real-world impact study has been measured. Recommendations are relative comparisons of forecast values, not health guarantees.

## Novelty

The prototype emphasizes transparent data handling: forecast values are joined to the challenge interval reference, observation and forecast intervals remain distinct, gaps are retained, and new plans are calculated only by Go. Offline behavior replays a previously saved result rather than creating a new recommendation without the Go service.

## Demonstrated functionality

- **IMPLEMENTED:** display observations and the selected forecast for four stations, with source and interval provenance.
- **IMPLEMENTED:** show an hourly chart, a tabular alternative, summary bands, and a near-term trajectory.
- **IMPLEMENTED:** calculate a lowest-forecast complete activity window in Go; an optional preferred start adds a comparison. The signed difference is the recommended mean minus the preferred mean.
- **IMPLEMENTED:** restore a validated offline snapshot and replay an exact saved plan without recalculation.
- **IMPLEMENTED:** English, Marathi, and Hindi interface resources and deterministic explanation fallbacks.
- **SIMULATED:** generate a forecast-derived Air Beacon preview and diagnostic pattern in memory; no device is controlled.
- **OPTIONAL:** a local English Gemma explanation passed the application's checks and was reviewed; failures use a labeled template.

## Architecture

The Go service loads and validates the pinned local CSV inputs, exposes typed HTTP endpoints, and owns all new activity calculations. The Angular application displays the service responses and uses a production service worker plus browser storage for offline assets, validated snapshots, and exact saved-result replay. The optional local explanation path uses Ollama when it produces validated output; otherwise, the application reports and uses a deterministic template. Beacon output is an in-memory simulation.

## ClimateVerse role

The source configuration records ClimateVerse catalogue metadata (`10.71646/XXOBKO`, version `1.0`) as metadata-discovery provenance and lists Open-Meteo as the data provider. This is citation/provenance metadata, not a runtime ClimateVerse or weather-service connection. Weather context is disabled and **UNAVAILABLE**.

## Forecasting-pipeline role

The application consumes the supplied `submission_xgb_corrected_scale_110.csv` forecast artifact. It does not retrain the model or run the competition's Python pipeline. The selected artifact already has 1.10 scaling applied; Air Pulse does not scale it again. The artifact's forecast generation time is unavailable.

## Google Gemma role

The application requests English explanations from the local `gemma4:e4b` model through Ollama. After earlier timeouts and invalid output, a revised prompt produced reviewed responses that passed the application's checks through the API and the explicit browser action. One API response took about 16 seconds, within the 30-second limit. These samples do not guarantee future model output; timeouts or failed checks use a labeled template. Deterministic English, Marathi, and Hindi explanations remain available.

## Go role

Go 1.27.4 is the local data-validation and API runtime. It computes activity recommendations and optional preferred-time comparisons, serves source/capability status, and derives simulated beacon payloads from validated forecast data. The browser does not implement a second planner.

## Angular/PWA role

Angular 22 provides the localized Air Pulse interface, forms, chart/table views, and explicit explanation and beacon actions. The production service worker caches application assets; browser storage supports offline snapshot restoration and exact saved-result replay. The user validated both the native share flow, including the operating-system share sheet, and copying the plan summary; the copied text was pasted successfully.

## Arduino role

**SIMULATED:** the browser shows a static forecast-derived color marker and pattern description. A bounded v1 payload codec is documented and tested in Go. Arduino firmware, serial transport, device acknowledgment, and physical-device behavior are **PLANNED/OPTIONAL** and unverified.

The user's photographs in `docs\submission-assets\uno-q-powered-redacted-v2.jpg` (preferred, clearer) and `uno-q-powered-redacted.jpg` show a powered UNO Q. They are hardware/setup evidence only: the lights and LED-matrix pattern do not demonstrate control by Air Guardian. The radio-module label (including its MAC address and serial data) is obscured and camera metadata is removed.

A read-only Windows device-inventory check on 13 November 2026 detected the connected board as a USB composite device identifying as Uno Q; its CDC serial and ADB interfaces reported `OK`. This confirms host recognition only. No serial port was opened, no commands were sent, and no firmware was uploaded; Air Guardian-to-board communication and physical output remain unverified.

## Multilingual support

The interface and deterministic explanation templates support English, Marathi, and Hindi. The user reviewed the Marathi and Hindi introductions, activity-planning controls, results, and deterministic explanations for a 12 November 2026 plan. The user subsequently approved the corrected Marathi and Hindi preferred-time comparison and beacon wording, then gave overall language approval on 13 November 2026. Individual review of every notice and shared-plan preview was not separately observed. Gemma generation is English-only; a local English sample passed the application's checks and was reviewed.

## Responsible-AI safeguards

- The interface identifies the product as a fixed-dataset view and does not infer a forecast issue time from retrieval or device time.
- Explanations are requested only after an explicit user action; the Go-computed plan remains authoritative.
- Model output is validated before use; unavailable or invalid generation uses an identified deterministic fallback.
- Values are rendered as text, and the interface repeatedly states that comparisons are not medical advice or health guarantees.
- No live weather, sensor readings, or physical beacon state is implied.

## Provenance

Startup validates the following pinned inputs:

| Input | Rows | SHA-256 |
| --- | ---: | --- |
| `zindi_artifacts\Train.csv` | 48,400 | `8adbcf55315f166d7dc3dc1d3e4d7400651f26c1d8b95ff387eef38f4e70a283` |
| `zindi_artifacts\submission_xgb_corrected_scale_110.csv` | 2,592 | `f82875f5f6ce820fafea9e53ef93298f3a21c29d514bbad9caec260f17c907d4` |
| `zindi_artifacts\Test.csv` | 2,592 | `e7b9cc8859beb939276c958393e148adae55229c34f99dcb2555fe4ac13b28db` |
| `zindi_artifacts\Stations.csv` | 4 | `383269fa04abf8f2eb49e5ec0bb5a77c2df11acb6aca996b7edf269a109ee9cf` |

Observation and forecast timestamp semantics are documented in [`TIMESTAMP_SEMANTICS.md`](TIMESTAMP_SEMANTICS.md). All 85 files in the supplied `zindi_artifacts` directory retain the recorded aggregate fingerprint `83adadccdfd2155e59abbf0b77053873a1c9982710728e6d981c75861123845b`.

## Limitations

- The app uses a fixed local dataset, not live monitoring or current readings.
- Forecast issuance time is not present in the input data.
- Data coverage is limited to the four supplied stations and forecast intervals.
- Weather context is disabled; no external weather feed is validated.
- Gemma may time out or return invalid output, and validation cannot guarantee every claim; Arduino firmware and physical transport are not implemented.
- Native share flow and clipboard copy/paste were validated by the user.
- The user approved language and accessibility acceptance. Detailed manual assistive-technology coverage was not independently observed, and 28 automated contrast checks remain inconclusive; the UI has not been declared fully accessible.
- No public deployment, repository URL, final PDF, or project submission is included.

## Demo instructions

Follow the Windows setup and run instructions in [`..\README.md`](..\README.md). Start the Go API and Angular frontend locally, then open the documented `http://localhost:4200/` origin. The local Go service is required for new plans; when unavailable, the PWA can replay only an exact saved plan.

## Screenshot list

The following locally captured evidence is available in `docs\submission-assets`:

| File | Evidence |
| --- | --- |
| `air-pulse-overview.png` | Air Pulse overview and fixed-snapshot labels. |
| `activity-plan-gemma.png` | Go-backed plan, preferred-time comparison, share preview and a reviewed local English Gemma response. |
| `saved-plan-offline-replay.png` | Exact saved-plan replay without recalculation during a browser-offline check. |
| `beacon-simulation.png` | Forecast-derived visual preview labeled simulation only. |
| `uno-q-powered-redacted-v2.jpg` | Preferred, clearer photograph of the user's powered UNO Q; module label redacted; hardware/setup evidence only, not a demonstration of app control. |
| `uno-q-powered-redacted.jpg` | Earlier UNO Q photograph; same limitations. |

`marathi-plan-review.png`, `hindi-plan-review.png`, `marathi-beacon-review.png` and `hindi-beacon-review.png` show the revised comparison and beacon wording, which the user approved on 13 November 2026.

## Two-minute judging pitch

Mumbai Air Guardian is a local prototype for exploring supplied air-quality data without confusing a fixed forecast with live conditions. Its Air Pulse interface presents observations and forecast intervals for four Mumbai stations, preserves missing intervals as gaps, and makes source and timing limitations visible. This matters because a timestamp embedded in a forecast identifier is not enough to determine what hour the value describes.

The application joins forecast values to the supplied interval reference and validates the selected 1.10-scaled artifact without scaling it again. When someone plans an activity, the Go service compares complete contiguous forecast windows for the selected station and returns a recommendation with its mean forecast PM2.5. A preferred start time is optional; without one, the app does not calculate or display a comparison metric.

The Angular PWA can restore a validated local snapshot and replay an exact saved plan if the Go service is unavailable. It never creates a new offline recommendation. The interface and deterministic explanation templates support English, Marathi, and Hindi; the user has reviewed the main planning, comparison, explanation and beacon flows and approved overall language acceptance.

Optional capabilities are labeled honestly. Explanations require an explicit request; a local English Gemma response has been demonstrated, with a deterministic fallback when generation fails validation or times out. Air Beacon is a forecast-derived visual simulation; no physical device is integrated with or controlled by the app. The contribution demonstrated here is careful interval interpretation, transparent provenance, and a reproducible local comparison—not a medical recommendation, a live-monitoring claim, or measured health impact.
