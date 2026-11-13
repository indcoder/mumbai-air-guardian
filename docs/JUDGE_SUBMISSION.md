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

Forecast numbers alone do not show which complete activity window has relatively lower predicted PM2.5, what evidence supports the recommendation, or whether someone can understand it in a language they use.

<div class="question-grid" aria-label="Three questions Air Guardian helps answer">
<article><p class="question-label">WHEN?</p><p>Which complete activity period has relatively lower predicted PM2.5?</p></article>
<article><p class="question-label">WHY?</p><p>What evidence supports the recommendation?</p></article>
<article><p class="question-label">UNDERSTANDABLE?</p><p>Can the result be explained in English, Marathi or Hindi?</p></article>
</div>

Air Guardian compares supplied forecast values; a lower value is not a claim that a period is safe or healthy.

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

Air Pulse distinguishes the latest supplied observation from the forecast peak, minimum and near-term trajectory.

<div class="outlook-crops" aria-label="Air Pulse evidence, enlarged from the application screenshot">
<figure>
<figcaption>Latest observation and forecast peak</figcaption>
<div class="outlook-crop"><img src="docs/submission-assets/air-outlook-cards.png" alt="Air Pulse cards enlarged: latest observation 189.93 micrograms per cubic metre, and forecast peak 255.8 micrograms per cubic metre"></div>
</figure>
<figure>
<figcaption>Forecast minimum and near-term trajectory</figcaption>
<div class="outlook-crop outlook-crop--right"><img src="docs/submission-assets/air-outlook-cards.png" alt="Air Pulse cards enlarged: forecast minimum 40.51 micrograms per cubic metre, and an improving near-term trajectory"></div>
</figure>
</div>

Forecast values are joined to the challenge interval reference; missing intervals remain gaps. Observation and forecast time ranges stay distinct, and the forecast issuance time is unavailable.

<div class="page-break"></div>

## Prediction is mathematical. Explanation is generative.

Google Gemma 4 (`gemma4:e4b`) is the local generative explanation layer, run through Ollama. A user explicitly requests an explanation of a plan already calculated by Go.

When local generation is unavailable, times out, or fails validation, Air Guardian uses a clearly labelled deterministic explanation fallback.

<div class="concept-pair" aria-label="Prediction is mathematical; explanation is generative">
<article><p class="concept-label">PREDICTION</p><p>Mathematical</p><p>Go calculates the activity period from supplied PM2.5 forecast values.</p></article>
<article><p class="concept-label">EXPLANATION</p><p>Generative</p><p>Gemma explains bounded evidence; it does not calculate PM2.5 or choose the period.</p></article>
</div>

PM2.5 comes from the supplied forecast; Go selects the period. Gemma receives bounded Go-computed evidence and explains only. English generation is currently verified locally; Marathi and Hindi use deterministic localized templates. A reviewed English response passed application checks; future output is not guaranteed to pass validation.

<div class="gemma-examples" aria-label="Enlarged application evidence for the Go plan and local Gemma explanation">
<figure><figcaption>Go-selected plan and comparison</figcaption><div class="gemma-crop gemma-crop--plan"><img src="docs/submission-assets/activity-plan-gemma.png" alt="Enlarged activity recommendation and preferred-time comparison from the local application screenshot"></div></figure>
<figure><figcaption>Local Gemma explanation for the selected plan</figcaption><div class="gemma-crop gemma-crop--explanation"><img src="docs/submission-assets/activity-plan-gemma.png" alt="Enlarged grounded explanation card from the local application screenshot"></div></figure>
</div>
<p class="screenshot-note">Screenshot shows a separate validated planning example.</p>

<div class="page-break"></div>

## Mumbai speaks three languages

The interface and deterministic explanation templates support English, Marathi and Hindi.

<p class="language-emphasis">Language access applies to planning controls, results and deterministic explanations—not just the landing page.</p>

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

The project owner reviewed the shown Marathi and Hindi planning and explanation flows. This is not professional linguistic certification; Gemma generation in these languages is not implemented.

<div class="page-break"></div>

## Beyond the screen

The Air Beacon simulation translates a forecast state into a static color marker and text-described pulse pattern. It explores an accessible ambient signal without implying a sensor or medical measurement.

<div class="beacon-status" aria-label="Air Beacon and Arduino UNO Q implementation boundaries">
<article><p class="status-label">AIR BEACON SIMULATION</p><p class="status-value">IMPLEMENTED</p></article>
<article><p class="status-label">PHYSICAL ARDUINO CONTROL</p><p class="status-value">NOT IMPLEMENTED</p></article>
<article><p class="status-label">ARDUINO UNO Q</p><p class="status-value">HARDWARE DIRECTION / PROTOTYPE TARGET</p></article>
</div>

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

No firmware was uploaded, no serial communication occurred, and the preview does not indicate physical LED control or PM2.5 sensing.

<div class="page-break"></div>

## How it works

### Forecast → Evidence → Explanation → Decision → Ambient Signal

<svg class="architecture-diagram" role="img" aria-labelledby="architecture-title architecture-description" viewBox="0 0 1000 700" xmlns="http://www.w3.org/2000/svg">
<title id="architecture-title">Mumbai Air Guardian conceptual architecture</title>
<desc id="architecture-description">ClimateVerse catalogue context records an Open-Meteo provider entry but is separate from the selected competition forecast. The forecast flows to the Go evidence and activity planner. Go can send bounded evidence to local Gemma for explanation only. Go and Gemma explanations are presented by the Angular PWA, which supports sharing, exact offline replay and a simulated Air Beacon.</desc>
<defs><marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="5" orient="auto"><path d="M0 0 L10 5 L0 10 Z" fill="#315d56"/></marker></defs>
<g fill="none" stroke="#315d56" stroke-width="3" marker-end="url(#arrow)">
<path d="M380 86 H490"/>
<path d="M300 250 H365"/>
<path d="M650 250 H715"/>
<path d="M500 320 V390"/>
<path d="M830 320 C830 360 700 370 635 405"/>
<path d="M500 485 C470 520 340 520 315 560"/>
<path d="M500 485 C530 520 660 520 685 560"/>
</g>
<g font-family="Segoe UI, sans-serif" fill="#17262b">
<rect x="20" y="30" width="360" height="112" rx="14" fill="#edf5ef" stroke="#7ea99a" stroke-width="2"/>
<text x="200" y="68" text-anchor="middle" font-size="26" font-weight="700">ClimateVerse</text>
<text x="200" y="99" text-anchor="middle" font-size="20">Dataset discovery / provenance context</text>
<text x="200" y="126" text-anchor="middle" font-size="17">Not the PM2.5 forecast source</text>
<rect x="490" y="30" width="490" height="112" rx="14" fill="#f3f1ea" stroke="#bcae84" stroke-width="2"/>
<text x="735" y="68" text-anchor="middle" font-size="24" font-weight="700">Project provenance record</text>
<text x="735" y="99" text-anchor="middle" font-size="18">Open-Meteo catalogue entry; underlying provider</text>
<text x="735" y="126" text-anchor="middle" font-size="17">No runtime ClimateVerse API</text>
<rect x="20" y="195" width="280" height="125" rx="14" fill="#f3f1ea" stroke="#bcae84" stroke-width="2"/>
<text x="160" y="235" text-anchor="middle" font-size="22" font-weight="700">Selected challenge</text>
<text x="160" y="264" text-anchor="middle" font-size="22" font-weight="700">PM2.5 forecast</text>
<text x="160" y="294" text-anchor="middle" font-size="17">Fixed artifact; issue time unavailable</text>
<rect x="365" y="195" width="285" height="125" rx="14" fill="#e5f1eb" stroke="#4d8875" stroke-width="2"/>
<text x="507" y="235" text-anchor="middle" font-size="24" font-weight="700">Go evidence + planner</text>
<text x="507" y="268" text-anchor="middle" font-size="18">Validates intervals and gaps</text>
<text x="507" y="296" text-anchor="middle" font-size="18">Calculates and selects the period</text>
<rect x="715" y="195" width="265" height="125" rx="14" fill="#edf5ef" stroke="#7ea99a" stroke-width="2"/>
<text x="847" y="235" text-anchor="middle" font-size="24" font-weight="700">Local Gemma 4</text>
<text x="847" y="268" text-anchor="middle" font-size="18">Grounded explanation</text>
<text x="847" y="296" text-anchor="middle" font-size="18">Does not calculate or choose</text>
<text x="680" y="185" text-anchor="middle" font-size="15">Bounded evidence</text>
<rect x="360" y="390" width="280" height="95" rx="14" fill="#dcebe5" stroke="#315d56" stroke-width="3"/>
<text x="500" y="430" text-anchor="middle" font-size="26" font-weight="700">Angular 22 PWA</text>
<text x="500" y="460" text-anchor="middle" font-size="18">Localized decision experience</text>
<rect x="150" y="555" width="330" height="100" rx="14" fill="#f3f1ea" stroke="#bcae84" stroke-width="2"/>
<text x="315" y="596" text-anchor="middle" font-size="23" font-weight="700">Sharing + offline</text>
<text x="315" y="626" text-anchor="middle" font-size="18">Exact saved-plan replay</text>
<rect x="520" y="555" width="330" height="100" rx="14" fill="#f3f1ea" stroke="#bcae84" stroke-width="2"/>
<text x="685" y="596" text-anchor="middle" font-size="23" font-weight="700">Air Beacon</text>
<text x="685" y="626" text-anchor="middle" font-size="18">Simulation only</text>
</g>
</svg>

Go owns forecast evidence and activity calculations. Gemma is an explanation-only branch; the Angular PWA also provides deterministic English, Marathi and Hindi templates when local generation is unavailable or invalid.

<div class="page-break"></div>

## Why this is creative

<div class="judging-grid" aria-label="Four judging criteria">
<article><h3>IMPACT</h3><p>Multilingual understanding and activity planning around supplied PM2.5 forecasts.</p></article>
<article><h3>NOVELTY</h3><p class="stage-chain">Forecast → Evidence → Explanation → Decision → Ambient Signal</p></article>
<article><h3>FUNCTIONALITY</h3><ul><li>Working Go planner and four-station Air Pulse</li><li>PWA snapshot and exact saved-plan replay</li><li>Sharing and verified local English Gemma generation</li><li>Deterministic fallback and simulated Air Beacon</li></ul></article>
<article><h3>GOOGLE TECHNOLOGY</h3><ul><li>Google Gemma 4</li><li>Angular 22</li><li>Modern Web Guidance 1.0</li></ul><p>See <a href="docs/MODERN_WEB_REVIEW.md">review scope and limitations</a> and the <a href="https://developer.chrome.com/blog/modern-web-guidance-v1">Chrome Developers announcement</a>.</p></article>
</div>

<aside class="climateverse-strip">
<h3>CLIMATEVERSE</h3>
<p>Project provenance records a ClimateVerse India catalogue entry for Open-Meteo (DOI 10.71646/XXOBKO, version 1.0), while the configuration correctly identifies Open-Meteo as the underlying data provider.</p>
<p>ClimateVerse provides the project's dataset-discovery and provenance context. It is not presented as the source of the PM2.5 forecast or as a runtime weather API.</p>
<p>Weather context that cannot be validated remains unavailable.</p>
</aside>

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
