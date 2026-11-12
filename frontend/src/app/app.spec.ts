import 'fake-indexeddb/auto';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { LocaleService } from './core/locale.service';
import { of, throwError } from 'rxjs';
import englishMessages from '../assets/i18n/en.json';
import hindiMessages from '../assets/i18n/hi.json';
import marathiMessages from '../assets/i18n/mr.json';
import { AirGuardianApi, BeaconStatusResponse } from './core/air-guardian-api';
import { OfflineStorage, SavedExplanation, SavedPlan } from './core/offline-storage';
import { App } from './app';

const offlineStorage = {
  saveSnapshot: vi.fn(() => Promise.resolve()),
  loadSnapshot: vi.fn(() => Promise.resolve(null)),
  loadLatestSnapshot: vi.fn(() => Promise.resolve(null)),
  savePlan: vi.fn(() => Promise.resolve()),
  loadPlan: vi.fn(() => Promise.resolve(null as SavedPlan | null)),
  saveExplanation: vi.fn(() => Promise.resolve()),
  loadExplanation: vi.fn(() => Promise.resolve(null as SavedExplanation | null)),
  clearAll: vi.fn(() => Promise.resolve()),
};

const beaconResponse = (
  request: Parameters<AirGuardianApi['simulateBeacon']>[0],
): BeaconStatusResponse => ({
  status: 'simulated',
  mode: 'simulation',
  reason: 'Virtual output only; no physical serial device is opened.',
  lastPayload: {
    version: 1,
    stationId: '11611',
    pm25: 26.4,
    band: 'low',
    trajectory: 'unavailable',
    peakHour: '2026-11-10T09:00:00+05:30',
    rgb: { red: 45, green: 106, blue: 79 },
    pattern: request.diagnostic ? 'diagnostic' : request.steadyOnly ? 'steady' : 'double-pulse',
    diagnostic: request.diagnostic,
    steadyOnly: request.steadyOnly,
  },
});

describe('App', () => {
  const station = {
    locationId: '11611',
    name: 'Malad',
    sensorId: 'sensor-1',
    latitude: 19.18,
    longitude: 72.84,
    timezone: 'Asia/Kolkata',
  };
  const flaggedObservation = {
    stationId: '11611',
    sensorId: 'sensor-1',
    intervalStart: '2026-11-10T08:00:00+05:30',
    intervalEnd: '2026-11-10T09:00:00+05:30',
    sourceTimestamp: '2026-11-10T09:00:00+05:30',
    availableAtSourceMtime: '2026-11-10T09:15:00+05:30',
    pm25: 1200,
    mean: 1200,
    count: 1,
    stationName: 'Malad',
    seriesId: 'series-1',
    flagHigh1000: true,
    sourceKind: 'observed' as const,
  };
  const riskBand = {
    name: 'low' as const,
    lower: 0,
    lowerInclusive: true,
    upper: 30,
    upperInclusive: true,
    color: '#2D6A4F',
    icon: 'circle-check',
    beaconPattern: 'steady',
  };
  const api = {
    getStations: vi.fn(() => of({ stations: [station] })),
    getSources: vi.fn(() =>
      of({
        snapshotLabel: 'Challenge forecast snapshot',
        latestObservationLabel: 'Latest observation in supplied dataset',
        forecastGenerationTime: 'Forecast generation time unavailable',
        manifest: {
          timezone: 'Asia/Kolkata',
          forecastGeneratedAt: null,
          forecastGenerationStatus: 'unavailable',
          selectedForecastScalingAlreadyApplied: 1.1,
          sources: {
            observations: {
              kind: 'observed' as const,
              path: 'Train.csv',
              sha256: 'a'.repeat(64),
              rows: 48_400,
            },
            forecasts: {
              kind: 'forecast' as const,
              path: 'submission.csv',
              sha256: 'b'.repeat(64),
              rows: 2_592,
            },
            stations: {
              kind: 'reference' as const,
              path: 'Stations.csv',
              sha256: 'c'.repeat(64),
              rows: 4,
            },
          },
          climateVerseCatalogue: {
            doi: '10.71646/XXOBKO',
            version: '1.0',
            stage: 'planning-review',
            discoveredVia: 'ClimateVerse India Portal',
            catalogueRole: 'metadata-discovery-provenance',
            dataProvider: 'Open-Meteo',
          },
        },
      }),
    ),
    getAiStatus: vi.fn(() =>
      of({
        status: 'available',
        reason: 'The configured local model is installed.',
        model: 'gemma4:e4b',
        languages: ['en', 'mr', 'hi'],
        generatedLanguages: ['en'],
      }),
    ),
    simulateBeacon: vi.fn((request: Parameters<AirGuardianApi['simulateBeacon']>[0]) =>
      of(beaconResponse(request)),
    ),
    getHealth: vi.fn(() =>
      of({
        status: 'ok',
        capabilities: {
          observations: { status: 'available' },
          forecasts: { status: 'available' },
          weather: { status: 'unavailable', reason: 'not validated' },
          ai: { status: 'available', reason: 'Deterministic fallback is available' },
          beacon: { status: 'unavailable', reason: 'not implemented' },
        },
      }),
    ),
    getAirOutlook: vi.fn(() =>
      of({
        stationId: '11611',
        anchor: '2026-11-10T09:00:00+05:30',
        latestObservation: flaggedObservation,
        peak: { valuePm25: 26.4, band: riskBand },
        lowest: { valuePm25: 17.2, band: riskBand },
        trajectory: {
          status: 'unavailable' as const,
          pointsUsed: 1,
          partialCoverage: true,
          lookaheadHours: 6,
        },
        forecastGenerationTime: 'unavailable',
      }),
    ),
    getObservations: vi.fn(() =>
      of({
        stationId: '11611',
        from: '2026-10-11T09:00:00+05:30',
        to: '2026-11-10T09:00:00+05:30',
        points: [flaggedObservation],
        source: {
          kind: 'observed' as const,
          path: 'Train.csv',
          sha256: 'a'.repeat(64),
          rows: 48_400,
        },
        excludedFlaggedCount: 1,
        exclusionReason: 'Explicit source quality flag',
      }),
    ),
    getForecast: vi.fn(() =>
      of({
        stationId: '11611',
        from: '2026-11-10T09:00:00+05:30',
        to: '2026-11-24T09:00:00+05:30',
        points: [
          {
            id: 'forecast-1',
            stationId: '11611',
            sensorId: 'sensor-1',
            intervalStart: '2026-11-10T09:00:00+05:30',
            intervalEnd: '2026-11-10T10:00:00+05:30',
            pm25: 26.4,
            sourceKind: 'forecast' as const,
            sourceRow: 1,
          },
        ],
        source: {
          kind: 'forecast' as const,
          path: 'submission.csv',
          sha256: 'b'.repeat(64),
          rows: 2_592,
        },
        generatedAt: null,
        generationStatus: 'unavailable',
      }),
    ),
    createActivityPlan: vi.fn(() =>
      of({
        stationId: '11611',
        activityType: 'outdoor exercise',
        recommendedPeriod: {
          start: '2026-11-10T15:00:00+05:30',
          end: '2026-11-10T17:00:00+05:30',
          meanPm25: 52.5801715,
          coverageHours: 2,
        },
        resolutionMinutes: 60,
        sourceForecastSha256: 'b'.repeat(64),
        forecastGenerationTime: 'unavailable',
      }),
    ),
    createExplanation: vi.fn(() =>
      of({
        mode: 'template' as const,
        reasonCode: 'generation_timeout',
        evidenceHash: 'd'.repeat(64),
        explanation: {
          summary: 'The suggested period has the lowest mean among complete windows.',
          keySignals: ['Complete hourly forecast coverage.'],
          recommendedAction: 'Use only as a relative forecast comparison.',
          limitations: 'Fixed snapshot; not a health guarantee.',
          language: 'en',
        },
      }),
    ),
  };
  const localeMessages: Record<string, unknown> = {
    '/assets/i18n/en.json': englishMessages,
    '/assets/i18n/mr.json': marathiMessages,
    '/assets/i18n/hi.json': hindiMessages,
  };
  const localeHttp = {
    get: vi.fn((url: string) => of(localeMessages[url])),
  };

  beforeEach(async () => {
    vi.clearAllMocks();
    offlineStorage.loadPlan.mockResolvedValue(null);
    localeHttp.get.mockImplementation((url: string) => of(localeMessages[url]));
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        { provide: AirGuardianApi, useValue: api },
        { provide: OfflineStorage, useValue: offlineStorage },
        { provide: HttpClient, useValue: localeHttp },
      ],
    }).compileComponents();
    await TestBed.inject(LocaleService).initialize();
  });

  it('shows fixed-snapshot provenance and retains flagged observations as diagnostics', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('h1')?.textContent).toContain('Air Pulse');
    expect((compiled.querySelector('#earliestStart') as HTMLSelectElement).value).toBe('06:00');
    expect((compiled.querySelector('#latestEnd') as HTMLSelectElement).value).toBe('20:00');
    expect((compiled.querySelector('#durationMinutes') as HTMLSelectElement).value).toBe('120');
    expect(compiled.textContent).toContain('Challenge forecast snapshot');
    expect(compiled.textContent).toContain('Forecast generation time unavailable');
    expect(compiled.textContent).toContain('Source-flagged · excluded from summary');
    expect(compiled.textContent).toContain('1,200 µg/m³');
    expect(compiled.textContent).toContain('Beacon output is a virtual simulation only');
    expect(api.createExplanation).not.toHaveBeenCalled();
    expect(compiled.querySelector('.metric-card')?.textContent).not.toContain('Low band');
  });

  it('runs beacon previews only on request and labels simulated steady and diagnostic modes', async () => {
    api.simulateBeacon.mockImplementation((request) => of(beaconResponse(request)));
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.simulateBeacon).not.toHaveBeenCalled();
    const buttons = fixture.nativeElement.querySelectorAll(
      '.beacon-actions button',
    ) as NodeListOf<HTMLButtonElement>;
    buttons[0].click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.simulateBeacon).toHaveBeenNthCalledWith(1, {
      stationId: '11611',
      steadyOnly: false,
      diagnostic: false,
    });
    expect(fixture.nativeElement.textContent).toContain('Two gentle pulses every 4 seconds');
    expect(fixture.nativeElement.textContent).toContain(
      'Simulated only — Air Guardian does not control a physical device.',
    );

    (fixture.nativeElement.querySelector('#beacon-steady-only') as HTMLInputElement).click();
    fixture.detectChanges();
    buttons[0].click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.simulateBeacon).toHaveBeenNthCalledWith(2, {
      stationId: '11611',
      steadyOnly: true,
      diagnostic: false,
    });
    expect(fixture.nativeElement.textContent).toContain('Steady light');

    buttons[1].click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.simulateBeacon).toHaveBeenNthCalledWith(3, {
      stationId: '11611',
      steadyOnly: true,
      diagnostic: true,
    });
    expect(fixture.nativeElement.textContent).toContain('Steady diagnostic light');
    expect(fixture.nativeElement.textContent).toContain(
      'This diagnostic changes only the preview',
    );
  });

  it('omits preferred-time comparison when the user leaves that option unset', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const form = fixture.nativeElement.querySelector('form') as HTMLFormElement;
    expect((fixture.nativeElement.querySelector('#earliestStart') as HTMLSelectElement).value).toBe(
      '06:00',
    );
    expect((fixture.nativeElement.querySelector('#latestEnd') as HTMLSelectElement).value).toBe(
      '20:00',
    );
    form.requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.createActivityPlan).toHaveBeenCalledWith({
      stationId: '11611',
      activityType: 'outdoor exercise',
      date: '2026-11-10',
      earliestStart: '06:00',
      latestEnd: '20:00',
      durationMinutes: 120,
    });
    expect(fixture.nativeElement.textContent).toContain('Suggested activity period');
    expect(fixture.nativeElement.textContent).not.toContain('Preferred-time comparison');
    expect(api.createExplanation).not.toHaveBeenCalled();
  });

  it('labels the signed difference as recommended minus preferred in the result and share preview', async () => {
    api.createActivityPlan.mockReturnValueOnce(
      of({
        stationId: '11611',
        activityType: 'outdoor exercise',
        recommendedPeriod: {
          start: '2026-11-10T15:00:00+05:30',
          end: '2026-11-10T17:00:00+05:30',
          meanPm25: 52.58,
          coverageHours: 2,
        },
        comparison: {
          preferredPeriod: {
            start: '2026-11-10T09:00:00+05:30',
            end: '2026-11-10T11:00:00+05:30',
            meanPm25: 76.65,
            coverageHours: 2,
          },
          differencePm25: -24.07,
        },
        resolutionMinutes: 60,
        sourceForecastSha256: 'b'.repeat(64),
        forecastGenerationTime: 'unavailable',
      }),
    );
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const preferred = fixture.nativeElement.querySelector('#preferredStartTime') as HTMLSelectElement;
    preferred.value = '09:00';
    preferred.dispatchEvent(new Event('change'));
    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();

    const comparison = fixture.nativeElement.querySelector('.comparison-note') as HTMLElement;
    const preview = fixture.nativeElement.querySelector('.share-preview pre') as HTMLElement;
    expect(comparison.textContent).toContain('Recommended mean minus preferred mean: -24.07 µg/m³');
    expect(preview.textContent).toContain('Recommended mean minus preferred mean: -24.07 µg/m³');
    expect(preview.textContent).toContain('10 Nov 2026, 9:00 am – 10 Nov 2026, 11:00 am IST');
  });

  it('updates the saved-plan status when the displayed language changes', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();

    const status = fixture.nativeElement.querySelector('.plan-result .offline-status') as HTMLElement;
    expect(status.textContent?.trim()).toBe(englishMessages['offline.planSaved']);

    const language = fixture.nativeElement.querySelector('#language-choice') as HTMLSelectElement;
    language.value = 'mr';
    language.dispatchEvent(new Event('change'));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(status.textContent?.trim()).toBe(marathiMessages['offline.planSaved']);

    language.value = 'hi';
    language.dispatchEvent(new Event('change'));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(status.textContent?.trim()).toBe(hindiMessages['offline.planSaved']);
  });

  it('updates the saved-plan failure status when the displayed language changes', async () => {
    offlineStorage.savePlan.mockRejectedValueOnce(new Error('Storage unavailable'));
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();

    await TestBed.inject(LocaleService).select('hi');
    fixture.detectChanges();
    const status = fixture.nativeElement.querySelector('.plan-result .offline-status') as HTMLElement;
    expect(status.textContent?.trim()).toBe(
      hindiMessages['offline.planSaveError'].replace('{detail}', 'Storage unavailable'),
    );
  });

  it('requests an explanation only after the explicit button is activated', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();

    const explainButton = fixture.nativeElement.querySelector(
      '.explanation-action button',
    ) as HTMLButtonElement;
    expect(explainButton).toBeTruthy();
    expect(api.createExplanation).not.toHaveBeenCalled();

    explainButton.click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.createExplanation).toHaveBeenCalledWith(
      {
        stationId: '11611',
        activityType: 'outdoor exercise',
        date: '2026-11-10',
        earliestStart: '06:00',
        latestEnd: '20:00',
        durationMinutes: 120,
      },
      'en',
    );
    expect(fixture.nativeElement.textContent).toContain('Deterministic template');
    expect(fixture.nativeElement.textContent).toContain(
      'Fixed snapshot; not a health guarantee.',
    );
  });

  it('localizes the interface and requests explanations in the selected language', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    await TestBed.inject(LocaleService).select('mr');
    fixture.detectChanges();
    expect(document.documentElement.lang).toBe('mr');
    expect(fixture.nativeElement.textContent).toContain('मुंबईतील केंद्रे');
    expect(fixture.nativeElement.textContent).toContain('एअर बीकन पूर्वावलोकन');

    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.explanation-action button') as HTMLButtonElement).click();
    await fixture.whenStable();

    expect(api.createExplanation).toHaveBeenCalledWith(
      {
        stationId: '11611',
        activityType: 'outdoor exercise',
        date: '2026-11-10',
        earliestStart: '06:00',
        latestEnd: '20:00',
        durationMinutes: 120,
      },
      'mr',
    );
  });

  it.each([0, 502, 503, 504])(
    'replays an exact saved plan when the local Go service is unavailable (%i)',
    async (failureStatus) => {
    const savedResult = {
      stationId: '11611',
      activityType: 'outdoor exercise',
      recommendedPeriod: {
        start: '2026-11-10T15:00:00+05:30',
        end: '2026-11-10T17:00:00+05:30',
        meanPm25: 52.5801715,
        coverageHours: 2,
      },
      resolutionMinutes: 60,
      sourceForecastSha256: 'b'.repeat(64),
      forecastGenerationTime: 'unavailable',
    };
    api.createActivityPlan.mockImplementationOnce(() =>
      throwError(() => new HttpErrorResponse({ status: failureStatus })),
    );
    offlineStorage.loadPlan.mockResolvedValueOnce({
      schemaVersion: 2,
      key: 'exact inputs',
      savedAt: '2026-11-12T12:00:00.000Z',
      request: {
        stationId: '11611',
        activityType: 'outdoor exercise',
        date: '2026-11-10',
        earliestStart: '06:00',
        latestEnd: '20:00',
        durationMinutes: 120,
      },
      result: savedResult,
    });

    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.createActivityPlan).toHaveBeenCalledTimes(1);
    expect(fixture.nativeElement.textContent).toContain(
      'Saved plan from',
    );
    expect(fixture.nativeElement.textContent).toContain('52.58 µg/m³');

    await TestBed.inject(LocaleService).select('mr');
    fixture.detectChanges();
    const savedTime = new Intl.DateTimeFormat('mr-IN', {
      timeZone: 'Asia/Kolkata',
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date('2026-11-12T12:00:00.000Z'));
    const status = fixture.nativeElement.querySelector('.plan-result .offline-status') as HTMLElement;
    expect(status.textContent?.trim()).toBe(
      marathiMessages['offline.planReplay'].replace('{time}', savedTime),
    );
    },
  );

  it('replays a matching saved explanation after a gateway timeout', async () => {
    const savedExplanation = {
      mode: 'template' as const,
      evidenceHash: 'd'.repeat(64),
      explanation: {
        summary: 'Saved explanation for this exact plan.',
        keySignals: ['Complete hourly forecast coverage.'],
        recommendedAction: 'Use only as a relative forecast comparison.',
        limitations: 'Fixed snapshot; not a health guarantee.',
        language: 'en',
      },
    };
    api.createExplanation.mockImplementationOnce(() =>
      throwError(() => new HttpErrorResponse({ status: 504 })),
    );
    offlineStorage.loadExplanation.mockResolvedValueOnce({
      schemaVersion: 2,
      key: 'exact inputs',
      savedAt: '2026-11-12T12:00:00.000Z',
      request: {
        stationId: '11611',
        activityType: 'outdoor exercise',
        date: '2026-11-10',
        earliestStart: '06:00',
        latestEnd: '20:00',
        durationMinutes: 120,
      },
      language: 'en',
      sourceForecastSha256: 'b'.repeat(64),
      result: savedExplanation,
    });

    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.explanation-action button') as HTMLButtonElement).click();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(api.createExplanation).toHaveBeenCalledTimes(1);
    expect(fixture.nativeElement.textContent).toContain('Saved explanation for this exact plan.');
  });

  it('associates deferred inline validation with the invalid field', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const dateInput = fixture.nativeElement.querySelector('#planDate') as HTMLInputElement;
    dateInput.value = '';
    dateInput.dispatchEvent(new Event('input'));
    dateInput.dispatchEvent(new FocusEvent('blur'));
    fixture.detectChanges();

    expect(dateInput.getAttribute('aria-invalid')).toBe('true');
    expect(dateInput.getAttribute('aria-describedby')).toContain('date-error');
    expect(fixture.nativeElement.querySelector('#date-error')?.textContent).toContain(
      'Choose a value',
    );

    (fixture.nativeElement.querySelector('form') as HTMLFormElement).requestSubmit();
    fixture.detectChanges();
    expect(document.activeElement).toBe(dateInput);
  });
});
