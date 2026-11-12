# Modern Web Guidance review

## Scope and status

This focused review covers the Air Pulse interface, activity-planning flow, beacon simulation, and English/Marathi/Hindi locale switching in `frontend/src/index.html`, `frontend/src/app/app.html`, `app.ts`, `app.scss`, locale resources/services, and their component tests. It is an implementation review, not a claim of full WCAG conformance or a substitute for manual assistive-technology testing.

The installed Modern Web Guidance CLI is version 1.0.0. The review retrieved `required-field-feedback`, `validate-input-after-interaction`, `accessible-error-announcement`, `accessibility`, `responsive-table` and `responsive-design` using skill version `2026_11_09-5bcc021b`.

## Applied guidance

### Structure, names and focus

- The page starts with an English document title/language, one main landmark and one page-level heading. Selecting a locale updates the document's `lang` and title.
- The visible, explicitly labelled native language selector keeps language switching keyboard-operable. Marathi and Hindi use static locale resources with key/placeholder parity checks; there is no runtime translation.
- The skip link is the first focusable control and targets `main#main-content`, which has `tabindex="-1"` so focus can move to the destination without adding it to normal tab order.
- The station selector and each planner control have explicit label associations. Native inputs, selects, buttons, headings, disclosure elements and landmarks provide their built-in semantics.
- The SVG locator and chart have accessible titles/descriptions; the chart also has a text caption and semantic data-table alternative.

### Required fields and errors

- Required planner fields retain native `required` constraints and have visible asterisks plus a legend. The optional preferred start is not marked required.
- Angular field errors appear after blur or a failed submission rather than on initial render. `aria-invalid` and `aria-describedby` are updated together, and the error text explains the correction.
- Submission calls native `reportValidity()` and explicitly focuses the first invalid input/select. One polite announcement region reports submission status; individual error messages are not separate live regions.
- Native constraint validation and explicit event-driven state are used instead of depending on CSS `:user-invalid`; this avoids an unsupported-selector fallback and keeps the status logic testable. `aria-required="true"` is retained alongside `required` to satisfy the project's explicit accessibility instruction, although native `required` already conveys the state.
- Saved-plan, failure and replay notices use a polite status region and are rendered in the selected language; replay timestamps use the selected locale.

### Data presentation and responsive behavior

- The interval table uses a caption, column headers with `scope="col"` and row headers with `scope="row"`. It sits inside a labelled, keyboard-focusable scrolling region.
- The table keeps its semantic row/column structure and allows horizontal scrolling on narrow screens instead of converting cells into generated-content labels. This preserves the same header relationships across viewport sizes and avoids relying on CSS-generated text for essential labels.
- Layouts use viewport breakpoints, flexible grids, bounded `clamp()` typography and scrollable chart/table regions. Reduced-motion preferences are respected. No external font or script CDN is required.
- Outlook cards share content-sized label, value, status and detail rows using CSS subgrid, with explicit grid rows as a fallback. The observation band precedes its timestamp in both DOM and visual order, matching the forecast cards without fixed-height text or clipped translations.
- Date/time and numeric display uses `Intl` with the selected Indian locale while retaining the source's `Asia/Kolkata` timezone and unchanged Go planner inputs.
- Beacon actions are explicit native buttons; the steady-only option has a visible label, a 44px minimum label target, and separately associated help text. The color marker is static and decorative while localized text conveys the pattern, avoiding persistent animation rather than relying on a global reduced-motion override.
- The beacon panel shares the other panels' responsive padding while retaining its existing section and control gaps, keeping headings, the simulation badge, controls and preview results inset from the container border.

## Verification completed

- Angular production build passed with the user-local Node 22.23.5 runtime.
- Angular unit tests passed: 4 files, 26 tests, including locale-key/placeholder parity, signed comparison and share text, localized saved-plan states, accessible error association and invalid-field focus, and user-triggered beacon preview/diagnostic behavior.
- The live local preview loaded the supplied source snapshot and switched among English, Marathi and Hindi. Browser inspection confirmed localized headings, labels, time digits, document language/title, source labels, and offline status; no forecast/planner inputs changed during locale switching.
- The production build includes and service-worker-caches all three locale resources. Production-browser testing verified service-worker control, offline reload, replay of an exact saved plan without recalculation, and rejection of a different unsaved request. Stubbed clipboard success/failure and native-share success/cancellation/failure paths showed the expected status messages. The user subsequently validated native sharing and clipboard copy/paste.
- At a 320 CSS-pixel viewport, the document did not overflow horizontally. In forced-colors mode the focused language selector had a visible 3px solid outline. These limited checks do not replace manual zoom, keyboard or high-contrast review.
- Axe-core 4.14.0 reported no violations in the checked English, Marathi and Hindi overview states or the English plan/explanation state. Each audit left 28 color-contrast items incomplete. Representative muted text measured 6.04:1 against the paper color and approximately 5.76:1 against the strongest gold tint; those samples do not resolve every incomplete item.
- Production-browser smoke verified the forecast-derived beacon preview and explicit simulation/no-physical-control wording. Earlier smoke checks also covered the steady-only and diagnostic controls and static marker. The user approved the revised Marathi/Hindi beacon and comparison wording.
- After correcting the outlook-card alignment, Edge checks covered 51 station/locale/viewport combinations across all four stations and all three languages, including 320 CSS pixels. Pills and values aligned across cards sharing a row, with no overlapping content or card/page overflow.

## User acceptance and audit limitations

On 13 November 2026 the user explicitly approved accessibility and language acceptance. This records user sign-off, not an independently observed pass of every keyboard, screen-reader, zoom or high-contrast procedure. The 28 incomplete contrast checks remain an audit limitation, not a confirmed violation or a resolved conformance finding. Further manual testing with representative assistive technologies and Accessibility Insights is recommended; automated checks, sampled ratios and user acceptance do not establish WCAG conformance.

The user reviewed the Marathi and Hindi Air Pulse introductions and confirmed they read well and convey that the dataset is fixed rather than live. The user also approved the Hindi and Marathi activity-planning controls, result, deterministic template explanation and its limitations for a plan dated 12 November 2026, followed by the corrected comparison and beacon wording. Their subsequent overall language approval is recorded separately from those individually observed flows; it does not imply a documented line-by-line review of every notice or shared-plan preview.

## Explicit explanation action

The planner exposes a native `type="button"` only after Go returns a plan. The explanation endpoint is called only from that button's click handler; changes to station or planner inputs clear stale results and do not call the endpoint. Model-provided fields are rendered as Angular text bindings, while concentration values and plan calculations remain Go-owned. A concise visible `role="status"` region reports the asynchronous result, and request failures use a visible alert. This follows the forms guidance to “Announce AJAX submission results via `role="status"` or `aria-live`.”

The response clearly distinguishes validated Gemma text from the deterministic template fallback. Browser smoke testing covered a real-data plan and explicit requests returning both a fallback and, after the prompt revision, a reviewed local Gemma response. The checks do not guarantee semantic correctness; keyboard and screen-reader testing of this additional state remains part of final hardening.
