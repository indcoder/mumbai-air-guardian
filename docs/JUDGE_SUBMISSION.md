# Mumbai Air Guardian

## From air-quality forecasts to decisions people can understand and act on

**A multilingual air-quality decision companion for Mumbai**

**Public hackathon demo:** [https://air.hackmum.in](https://air.hackmum.in)<br>
**Source code and reproducibility:** [https://github.com/indcoder/mumbai-air-guardian](https://github.com/indcoder/mumbai-air-guardian)

Air Guardian turns the supplied challenge snapshot into Air Pulse: a way to inspect observations and forecasts, compare complete activity windows, and understand the result in English, Marathi or Hindi.

![Air Pulse overview showing the fixed-snapshot notice and four Mumbai monitoring stations](docs/submission-assets/air-pulse-overview.png)

*A fixed challenge-data snapshot, not live monitoring or medical advice.*

<div class="page-break"></div>

## The problem

Forecast numbers alone do not answer:

- What is expected in the supplied forecast?
- Which complete activity window has a relatively lower predicted PM2.5 value?
- Why did the planner select that period?
- Can someone understand the result in a language they use?

Air Guardian presents the supplied observations and forecast for four challenge stations. It distinguishes observed intervals from forecast intervals, preserves gaps instead of filling them, and keeps source and timing limitations visible.

The application does not claim that a lower forecast value makes a period safe or healthy. It is a transparent comparison of supplied forecast values.

<div class="page-break"></div>

## What Air Guardian actually does

### Malad West · 12 November 2026 · two-hour outdoor activity

<div class="metrics">
<div>
<p class="metric-label">PREFERRED PERIOD</p>
<p class="metric-period">08:00–10:00 IST</p>
<p class="metric-value">86.15</p>
<p>µg/m³ mean predicted PM2.5</p>
</div>
<div>
<p class="metric-label">LOWER-FORECAST OPTION</p>
<p class="metric-period">15:00–17:00 IST</p>
<p class="metric-value">52.58</p>
<p>µg/m³ mean predicted PM2.5</p>
</div>
<div>
<p class="metric-label">DIFFERENCE</p>
<p class="metric-value">33.57</p>
<p>µg/m³ lower among the available complete forecast windows</p>
</div>
</div>

Go calculates the recommendation deterministically from the supplied hourly forecast. Gemma may explain that result but cannot change forecast values or select the recommended period.

**Go calculates. Gemma explains.**

This example is source-backed by the regression test in the [repository](https://github.com/indcoder/mumbai-air-guardian/blob/main/backend/internal/activity/planner_test.go). Full precision and input hashes are in the [README provenance section](README.md#provenance).

<div class="page-break"></div>

## Air Pulse

Air Pulse shows the latest supplied observation, forecast peak and minimum, and near-term trajectory. Each period is presented with its time interval and a text risk band.

![Air Outlook cards showing latest observation, forecast peak, forecast minimum and near-term trajectory](docs/submission-assets/air-outlook-cards.png)

Forecast values are joined to the challenge interval reference; missing intervals remain gaps. Observation and forecast time ranges stay distinct. The forecast issuance time is not present in the supplied data.

<div class="page-break"></div>

## Prediction is mathematical. Explanation is generative.

Google Gemma 4 (`gemma4:e4b`) is the optional local explanation layer, run through Ollama. A user explicitly requests an explanation of a plan already calculated by Go.

- PM2.5 values come from the supplied forecast artifact.
- Go selects the relatively lower-forecast complete activity window.
- Gemma receives bounded evidence and explains the result; it never predicts PM2.5 or chooses a period.
- Invalid or timed-out model output is replaced with a labeled deterministic template.
- An English local Gemma response was generated and reviewed; future output is not guaranteed to pass validation.

![Activity plan with a reviewed local Gemma explanation, share preview and deterministic Go result](docs/submission-assets/activity-plan-gemma.png)

*Example UI state; this screenshot uses a different preferred period from the Malad West comparison on page 3.*

Gemma generation is currently English-only. Marathi and Hindi explanations use deterministic localized templates.

<div class="page-break"></div>

## Mumbai speaks three languages

The interface and deterministic explanation templates support English, Marathi and Hindi. Language access applies to the planning controls and results, not just the landing page.

<div class="visual-pair">
<div>
<h3>मराठी</h3>
<img src="docs/submission-assets/marathi-plan-review.png" alt="Activity planning controls, result and comparison shown in Marathi">
</div>
<div>
<h3>हिन्दी</h3>
<img src="docs/submission-assets/hindi-plan-review.png" alt="Activity planning controls, result and comparison shown in Hindi">
</div>
</div>

The author reviewed the shown planning and explanation flows in both languages and approved overall language acceptance. Gemma generation in Marathi and Hindi is not implemented.

<div class="page-break"></div>

## Beyond the screen

The Air Beacon simulation translates a forecast state into a static color marker and text-described pulse pattern. It explores an accessible ambient signal, without implying a sensor or medical measurement.

**Air Beacon:** a visual simulation, not physical-device control.
**UNO Q: host-recognized prototype target only**

<div class="visual-pair">
<div>
<h3>Air Beacon simulation</h3>
<img src="docs/submission-assets/beacon-simulation.png" alt="Air Beacon forecast-derived visual simulation labeled as simulation only">
</div>
<div>
<h3>Arduino UNO Q hardware direction</h3>
<img src="docs/submission-assets/uno-q-powered-redacted-v2.jpg" alt="Powered Arduino UNO Q board shown as hardware direction, not a device controlled by Air Guardian">
</div>
</div>

No firmware was uploaded, no serial communication occurred, and Air Guardian does not control the board.

<div class="page-break"></div>

## How it works

### Forecast → Evidence → Explanation → Decision → Ambient Signal

1. **Forecast artifact:** the selected challenge PM2.5 forecast provides quantitative values; its generation time is unavailable.
2. **Evidence:** Go validates the pinned inputs and joins forecasts to the supplied interval reference and stations.
3. **Decision:** one Go planner compares complete contiguous hourly windows. The browser does not implement a second planner.
4. **Explanation:** local Gemma can explain the Go-computed result on request; localized deterministic templates remain available as fallback.
5. **Experience:** Angular 22 presents the data and supports plan sharing plus exact saved-plan replay when offline.
6. **Ambient concept:** the Air Beacon preview derives a virtual pattern from validated forecast data.

The project records ClimateVerse catalogue metadata for dataset discovery and provenance; Open-Meteo is listed as the data provider. This metadata is separate from the forecast artifact and does not indicate a runtime ClimateVerse connection.

<div class="page-break"></div>

## Why this is creative

**Impact:** make a supplied PM2.5 forecast easier to understand and compare across activity windows, in three interface languages.

**Novelty:** connect the five stages—**Forecast → Evidence → Explanation → Decision → Ambient Signal**—instead of stopping at a chart.

**Functionality:** a deterministic planner, four-station Air Pulse, PWA snapshot and saved-plan replay, sharing, guarded local Gemma explanation, and simulated Air Beacon.

**Google technology in the working project:**

- **Gemma 4:** grounded, user-triggered local explanation.
- **Angular 22:** interactive multilingual PWA.
- **Modern Web Guidance 1.0:** informed the Angular frontend review workflow; see the [scope and limitations](docs/MODERN_WEB_REVIEW.md). Its stable release was [announced by Google's Chrome team on 10 November 2026](https://developer.chrome.com/blog/modern-web-guidance-v1).

**ClimateVerse:** project metadata records a discovery/provenance reference, with Open-Meteo listed as the data provider. It is not described as the source of the selected forecast or a runtime API.

<div class="page-break"></div>

## Try it · boundaries

| Demonstrated capability | What is not claimed |
| --- | --- |
| Air Pulse for four supplied stations | Live monitoring or hyperlocal coverage |
| Go activity planner and relative forecast comparison | A safe period, medical advice or health outcome |
| English, Marathi and Hindi interface and deterministic templates | Marathi or Hindi Gemma generation |
| Local English Gemma explanation, when Ollama is available | Gemma selecting the plan or changing forecast values |
| Offline restoration and replay of an exact saved plan | New recommendations while offline |
| Air Beacon visual simulation | Physical Arduino control or PM2.5 measurement |
| ClimateVerse metadata recorded for discovery/provenance | A ClimateVerse weather API or verified historical acquisition |

**Public hackathon demo:** [https://air.hackmum.in](https://air.hackmum.in)<br>
**Source code and reproducibility:** [https://github.com/indcoder/mumbai-air-guardian](https://github.com/indcoder/mumbai-air-guardian)<br>
The [README](README.md#run-locally-on-windows) contains local setup and verification instructions.

The public demo is provided for challenge evaluation and may be temporarily unavailable while deployment or maintenance is in progress, or restricted/taken offline in response to unexpected traffic or abuse. The GitHub repository remains the durable source and reproducibility artifact.

The challenge snapshot is fixed; its forecast issuance time is unavailable. Competition input files are not included in the repository because redistribution rights have not been confirmed. No behavior change or health impact has been measured.
