import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, DestroyRef, inject, OnInit, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  catchError,
  EMPTY,
  forkJoin,
  map,
  Observable,
  startWith,
  Subject,
  switchMap,
  tap,
} from 'rxjs';
import {
  ActivityPlanRequest,
  ActivityPlanResult,
  AiStatusResponse,
  AirGuardianApi,
  BeaconPayload,
  BeaconStatusResponse,
  AirOutlook,
  ExplanationResponse,
  ForecastPoint,
  HealthResponse,
  Observation,
  SourcesResponse,
  Station,
} from './core/air-guardian-api';
import { LocaleService, type LocaleCode } from './core/locale.service';
import { OfflineSnapshot, OfflineStorage } from './core/offline-storage';

interface DashboardData {
  outlook: AirOutlook;
  observations: Observation[];
  forecasts: ForecastPoint[];
  excludedFlaggedCount: number;
}

interface PlotPoint {
  start: number;
  end: number;
  time: number;
  value: number;
  label: string;
}

interface ChartPoint {
  x: number;
  y: number;
  description: string;
}

interface ChartSegment {
  id: string;
  kind: 'observation' | 'forecast';
  path: string;
  points: ChartPoint[];
}

interface ChartTick {
  x: number;
  label: string;
}

interface LocalizedMessage {
  key: string;
  values: Record<string, string | number>;
}

type SavedPlanStatus =
  | { kind: 'saved' }
  | { kind: 'save-error'; error: unknown }
  | { kind: 'replayed'; savedAt: string };

@Component({
  selector: 'app-root',
  styleUrl: './app.scss',
  templateUrl: './app.html',
})
export class App implements OnInit {
  private readonly api = inject(AirGuardianApi);
  private readonly offlineStorage = inject(OfflineStorage);
  protected readonly locale = inject(LocaleService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly stationSelection = new Subject<string>();

  protected readonly stations = signal<Station[]>([]);
  protected readonly selectedStationId = signal('');
  protected readonly outlook = signal<AirOutlook | null>(null);
  protected readonly observations = signal<Observation[]>([]);
  protected readonly forecasts = signal<ForecastPoint[]>([]);
  protected readonly excludedFlaggedCount = signal(0);
  protected readonly sources = signal<SourcesResponse | null>(null);
  protected readonly capabilityStatuses = signal<Record<string, string>>({});
  protected readonly aiStatus = signal<AiStatusResponse | null>(null);
  protected readonly health = signal<HealthResponse | null>(null);
  protected readonly beaconResult = signal<BeaconStatusResponse | null>(null);
  protected readonly beaconError = signal('');
  protected readonly beaconAnnouncement = signal('');
  protected readonly beaconLoading = signal(false);
  protected readonly beaconSteadyOnly = signal(false);
  protected readonly dashboardLoading = signal(true);
  protected readonly dashboardError = signal('');
  protected readonly planLoading = signal(false);
  protected readonly planError = signal('');
  protected readonly planResult = signal<ActivityPlanResult | null>(null);
  protected readonly planAnnouncement = signal('');
  protected readonly planRequest = signal<ActivityPlanRequest | null>(null);
  protected readonly explanationLoading = signal(false);
  protected readonly explanationError = signal('');
  protected readonly explanationResult = signal<ExplanationResponse | null>(null);
  protected readonly explanationAnnouncement = signal('');
  private readonly offlineStatusMessage = signal<LocalizedMessage | null>(null);
  protected readonly offlineStatus = computed(() => {
    const message = this.offlineStatusMessage();
    if (message === null) {
      return '';
    }
    if (message.key !== 'offline.snapshotSaved' && message.key !== 'offline.snapshotLoaded') {
      return this.t(message.key, message.values);
    }

    const dateValue = (key: string) => {
      const value = message.values[key];
      return typeof value === 'string' && value !== '' ? value : null;
    };
    const generatedAt = dateValue('generatedAt');
    return this.t(message.key, {
      ...message.values,
      observations: this.formatCoverage(
        dateValue('observationStart'),
        dateValue('observationEnd'),
      ),
      forecast: this.formatCoverage(dateValue('forecastStart'), dateValue('forecastEnd')),
      generated: generatedAt ? this.formatTime(generatedAt) : this.t('snapshot.issueUnavailable'),
      retrieved: this.formatTime(String(message.values['retrievedAt'])),
      error: this.t('app.networkUnavailable'),
    });
  });
  private readonly savedPlanStatusState = signal<SavedPlanStatus | null>(null);
  protected readonly savedPlanStatus = computed(() => {
    const status = this.savedPlanStatusState();
    if (status === null) {
      return '';
    }
    if (status.kind === 'saved') {
      return this.t('offline.planSaved');
    }
    if (status.kind === 'save-error') {
      return this.t('offline.planSaveError', { detail: this.errorMessage(status.error) });
    }
    return this.t('offline.planReplay', { time: this.formatTime(status.savedAt) });
  });
  protected readonly savedExplanationStatus = signal('');
  protected readonly shareStatus = signal('');
  protected readonly fieldErrors = signal<Record<string, string>>({});
  protected readonly planDate = signal('');
  protected readonly activityType = signal('outdoor exercise');
  protected readonly earliestStart = signal('06:00');
  protected readonly latestEnd = signal('20:00');
  protected readonly durationMinutes = signal('120');
  protected readonly preferredStartTime = signal('');
  private planRequestId = 0;
  private explanationRequestId = 0;
  private beaconRequestId = 0;

  protected readonly selectedStation = computed(
    () =>
      this.stations().find((station) => station.locationId === this.selectedStationId()) ?? null,
  );
  protected readonly sourceEntries = computed(() => {
    const records = this.sources()?.manifest.sources ?? {};
    const labels: Record<string, string> = {
      forecasts: 'source.forecasts',
      forecastIntervals: 'source.forecastReference',
      observations: 'source.observations',
      stations: 'source.stations',
    };
    return Object.entries(records).map(([name, record]) => ({
      name,
      label: labels[name] ? this.t(labels[name]) : name,
      record,
    }));
  });
  protected readonly stationMarkers = computed(() => {
    const stations = this.stations();
    if (stations.length === 0) {
      return [];
    }

    const latitudes = stations.map((station) => station.latitude);
    const longitudes = stations.map((station) => station.longitude);
    const minLatitude = Math.min(...latitudes);
    const latitudeSpan = Math.max(...latitudes) - minLatitude || 1;
    const minLongitude = Math.min(...longitudes);
    const longitudeSpan = Math.max(...longitudes) - minLongitude || 1;

    return stations.map((station) => ({
      ...station,
      x: 16 + ((station.longitude - minLongitude) / longitudeSpan) * 68,
      y: 16 + (1 - (station.latitude - minLatitude) / latitudeSpan) * 68,
      selected: station.locationId === this.selectedStationId(),
    }));
  });
  protected readonly chart = computed(() => this.createChart());
  protected readonly chartRows = computed(() => this.createChartRows());
  protected readonly timeOptions = Array.from(
    { length: 24 },
    (_, hour) => `${String(hour).padStart(2, '0')}:00`,
  );
  protected readonly endTimeOptions = [...this.timeOptions.slice(1), '24:00'];
  protected readonly durationOptions = [
    { value: '60', labelKey: 'form.durationOne' },
    { value: '120', labelKey: 'form.durationTwo' },
    { value: '180', labelKey: 'form.durationThree' },
  ];

  protected t(key: string, values: Record<string, string | number> = {}): string {
    return this.locale.text(key, values);
  }

  protected changeLanguage(event: Event): void {
    const target = event.target;
    if (target instanceof HTMLSelectElement) {
      void this.locale.select(target.value).then((changed) => {
        if (changed) {
          this.planAnnouncement.set('');
          this.explanationAnnouncement.set('');
          this.explanationResult.set(null);
          this.explanationError.set('');
          this.savedExplanationStatus.set('');
          this.shareStatus.set('');
          this.beaconAnnouncement.set('');
          this.beaconError.set('');
          this.fieldErrors.set({});
        }
      });
    }
  }

  ngOnInit(): void {
    forkJoin({
      stations: this.api.getStations(),
      sources: this.api.getSources(),
      health: this.api.getHealth(),
    })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: ({ stations, sources, health }) => {
          this.stations.set(stations.stations);
          this.sources.set(sources);
          this.health.set(health);
          this.capabilityStatuses.set(
            Object.fromEntries(
              Object.entries(health.capabilities).map(([name, capability]) => [
                name,
                capability.status,
              ]),
            ),
          );
          this.api
            .getAiStatus()
            .pipe(takeUntilDestroyed(this.destroyRef))
            .subscribe({
              next: (status) => {
                this.aiStatus.set(status);
                this.capabilityStatuses.update((capabilities) => ({
                  ...capabilities,
                  ai: status.status,
                }));
              },
              error: () => {
                this.capabilityStatuses.update((capabilities) => ({
                  ...capabilities,
                  ai: 'unavailable',
                }));
              },
            });
          const firstStation =
            stations.stations.find((station) => station.locationId === '11611') ??
            stations.stations[0];
          if (!firstStation) {
            this.dashboardLoading.set(false);
            this.dashboardError.set(this.t('app.noStations'));
            return;
          }

          this.selectedStationId.set(firstStation.locationId);
          this.stationSelection
            .pipe(
              startWith(firstStation.locationId),
              tap(() => {
                this.dashboardLoading.set(true);
                this.dashboardError.set('');
                this.clearPlanOutput();
                this.beaconRequestId += 1;
                this.beaconResult.set(null);
                this.beaconError.set('');
                this.beaconAnnouncement.set('');
                this.beaconLoading.set(false);
              }),
              switchMap((stationId) =>
                this.loadStationData(stationId).pipe(
                  tap(() => this.dashboardLoading.set(false)),
                  catchError((error: unknown) => {
                    void this.restoreSnapshot(stationId, this.errorMessage(error));
                    return EMPTY;
                  }),
                ),
              ),
              takeUntilDestroyed(this.destroyRef),
            )
            .subscribe((data) => this.applyDashboardData(data));
        },
        error: (error: unknown) => {
          void this.restoreLatestSnapshot(this.errorMessage(error));
        },
      });
  }

  protected onStationChange(event: Event): void {
    const stationId = (event.target as HTMLSelectElement).value;
    if (this.stations().some((station) => station.locationId === stationId)) {
      this.clearPlanOutput();
      this.beaconRequestId += 1;
      this.beaconResult.set(null);
      this.beaconError.set('');
      this.beaconAnnouncement.set('');
      this.beaconLoading.set(false);
      this.outlook.set(null);
      this.observations.set([]);
      this.forecasts.set([]);
      this.selectedStationId.set(stationId);
      this.stationSelection.next(stationId);
    }
  }

  protected refresh(): void {
    const stationId = this.selectedStationId();
    if (stationId !== '') {
      this.clearPlanOutput();
      this.beaconRequestId += 1;
      this.beaconResult.set(null);
      this.beaconError.set('');
      this.beaconAnnouncement.set('');
      this.beaconLoading.set(false);
      this.stationSelection.next(stationId);
    }
  }

  protected updateBeaconSteadyOnly(event: Event): void {
    const target = event.target;
    if (target instanceof HTMLInputElement) {
      this.beaconSteadyOnly.set(target.checked);
    }
  }

  protected updateFormValue(field: string, event: Event): void {
    const target = event.target;
    if (!(target instanceof HTMLInputElement || target instanceof HTMLSelectElement)) {
      return;
    }

    const value = target.value;
    this.clearPlanOutput();
    switch (field) {
      case 'planDate':
        this.planDate.set(value);
        break;
      case 'activityType':
        this.activityType.set(value);
        break;
      case 'earliestStart':
        this.earliestStart.set(value);
        break;
      case 'latestEnd':
        this.latestEnd.set(value);
        break;
      case 'durationMinutes':
        this.durationMinutes.set(value);
        break;
      case 'preferredStartTime':
        this.preferredStartTime.set(value);
        break;
    }

    const error = this.validationMessage(target);
    if (error === '') {
      this.fieldErrors.update((errors) => {
        const next = { ...errors };
        delete next[field];
        return next;
      });
    } else if (this.fieldErrors()[field]) {
      this.fieldErrors.update((errors) => ({ ...errors, [field]: error }));
    }
  }

  protected onFieldBlur(event: FocusEvent, field: string): void {
    const target = event.target;
    if (!(target instanceof HTMLInputElement || target instanceof HTMLSelectElement)) {
      return;
    }
    const error = this.validationMessage(target);
    if (error !== '') {
      this.fieldErrors.update((errors) => ({ ...errors, [field]: error }));
    } else {
      this.fieldErrors.update((errors) => {
        const next = { ...errors };
        delete next[field];
        return next;
      });
    }
  }

  protected fieldError(field: string): string {
    return this.fieldErrors()[field] ?? '';
  }

  protected submitPlan(event: SubmitEvent): void {
    event.preventDefault();
    if (this.planLoading()) {
      return;
    }
    this.clearPlanOutput();
    const form = event.currentTarget;
    if (!(form instanceof HTMLFormElement)) {
      return;
    }
    this.updateFormErrors(form);
    if (!form.reportValidity()) {
      form.querySelector<HTMLInputElement | HTMLSelectElement>(':invalid')?.focus();
      this.planAnnouncement.set(
        this.t('validation.formError'),
      );
      return;
    }

    const request: ActivityPlanRequest = {
      stationId: this.selectedStationId(),
      activityType: this.activityType(),
      date: this.planDate(),
      earliestStart: this.earliestStart(),
      latestEnd: this.latestEnd(),
      durationMinutes: Number(this.durationMinutes()),
      ...(this.preferredStartTime() !== ''
        ? { preferredStartTime: this.preferredStartTime() }
        : {}),
    };

    this.planLoading.set(true);
    this.planError.set('');
    this.planResult.set(null);
    this.planAnnouncement.set(this.t('plan.createProgress'));
    const requestId = ++this.planRequestId;
    this.api
      .createActivityPlan(request)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (requestId !== this.planRequestId) {
            return;
          }
          this.planLoading.set(false);
          this.planResult.set(result);
          this.planRequest.set(request);
          this.planAnnouncement.set(
            this.t('plan.createAnnouncement', {
              start: this.formatTime(result.recommendedPeriod.start),
              end: this.formatTime(result.recommendedPeriod.end),
            }),
          );
          void this.persistPlan(request, result, requestId);
        },
        error: (error: unknown) => {
          if (this.canReplaySavedResult(error)) {
            void this.restoreSavedPlan(request, requestId);
            return;
          }
          if (requestId !== this.planRequestId) {
            return;
          }
          this.planLoading.set(false);
          this.planError.set(this.errorMessage(error));
          this.planAnnouncement.set(this.t('plan.createError', { error: this.planError() }));
        },
      });
  }

  protected requestExplanation(): void {
    const request = this.planRequest();
    if (request === null || this.explanationLoading()) {
      return;
    }

    const requestId = ++this.explanationRequestId;
    this.explanationLoading.set(true);
    this.explanationError.set('');
    this.explanationResult.set(null);
    this.explanationAnnouncement.set(this.t('plan.explanationProgress'));
    const language = this.locale.current();
    this.api
      .createExplanation(request, language)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (requestId !== this.explanationRequestId) {
            return;
          }
          this.explanationLoading.set(false);
          this.explanationResult.set(result);
          this.explanationAnnouncement.set(
            result.mode === 'gemma'
              ? this.t('plan.explanationGemmaReady')
              : this.t('plan.explanationTemplateReady'),
          );
          const sourceForecastSha256 = this.planResult()?.sourceForecastSha256 ?? '';
          void this.persistExplanation(request, language, sourceForecastSha256, result, requestId);
        },
        error: (error: unknown) => {
          if (this.canReplaySavedResult(error)) {
            void this.restoreSavedExplanation(request, requestId);
            return;
          }
          if (requestId !== this.explanationRequestId) {
            return;
          }
          this.explanationLoading.set(false);
          this.explanationError.set(this.errorMessage(error));
          this.explanationAnnouncement.set(
            this.t('plan.explanationError', { error: this.explanationError() }),
          );
        },
      });
  }

  protected formatConcentration(value: number): string {
    return `${new Intl.NumberFormat(this.locale.localeTag(), {
      maximumFractionDigits: 2,
    }).format(value)} µg/m³`;
  }

  protected formatNumber(value: number): string {
    return new Intl.NumberFormat(this.locale.localeTag(), {
      maximumFractionDigits: 2,
    }).format(value);
  }

  protected formatSignedNumber(value: number): string {
    return `${value > 0 ? '+' : ''}${this.formatNumber(value)}`;
  }

  protected formatClockLabel(value: string): string {
    const digits = new Intl.NumberFormat(this.locale.localeTag(), {
      useGrouping: false,
      maximumFractionDigits: 0,
    });
    return value.replace(/\d/g, (digit) => digits.format(Number(digit)));
  }

  protected formatCoordinate(value: number): string {
    return new Intl.NumberFormat(this.locale.localeTag(), {
      maximumFractionDigits: 4,
    }).format(value);
  }

  protected forecastIssueTime(): string {
    const generatedAt = this.sources()?.manifest.forecastGeneratedAt;
    return generatedAt ? this.formatTime(generatedAt) : this.t('intro.issueUnavailable');
  }

  protected formatTime(value: string): string {
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      return this.t('metric.unavailable');
    }
    return new Intl.DateTimeFormat(this.locale.localeTag(), {
      timeZone: 'Asia/Kolkata',
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(parsed);
  }

  private formatCoverage(start: string | null, end: string | null): string {
    return start !== null && end !== null
      ? `${this.formatTime(start)} – ${this.formatTime(end)} IST`
      : this.t('status.unavailable');
  }

  protected formatShortTime(value: number): string {
    return new Intl.DateTimeFormat(this.locale.localeTag(), {
      timeZone: 'Asia/Kolkata',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(value));
  }

  protected riskLabel(name: string): string {
    switch (name) {
      case 'low':
        return this.t('risk.low');
      case 'elevated':
        return this.t('risk.elevated');
      case 'high':
        return this.t('risk.high');
      case 'very_high':
        return this.t('risk.veryHigh');
      default:
        return this.t('risk.unclassified');
    }
  }

  protected trajectoryLabel(status: string): string {
    switch (status) {
      case 'improving':
        return this.t('trajectory.improving');
      case 'stable':
        return this.t('trajectory.stable');
      case 'worsening':
        return this.t('trajectory.worsening');
      default:
        return this.t('trajectory.unavailable');
    }
  }

  protected capabilityStatus(name: string): string {
    const status = this.capabilityStatuses()[name] ?? 'unknown';
    return this.t(
      status === 'available' || status === 'unavailable' || status === 'unknown'
        ? `status.${status}`
        : status === 'simulated'
          ? 'status.simulated'
          : 'status.unknown',
    );
  }

  protected explanationModeLabel(result: ExplanationResponse): string {
    return result.mode === 'gemma'
      ? this.t('plan.modeGemma')
      : this.t('plan.modeTemplate');
  }

  protected activityLabel(activityType: string): string {
    switch (activityType) {
      case 'outdoor exercise':
        return this.t('form.outdoorExercise');
      case 'walk':
        return this.t('form.walk');
      case 'commute':
        return this.t('form.commute');
      default:
        return activityType;
    }
  }

  protected async copyPlanSummary(): Promise<void> {
    const text = this.planSummaryText();
    if (text === '') {
      return;
    }
    try {
      await navigator.clipboard.writeText(text);
      this.shareStatus.set(this.t('share.copied'));
    } catch {
      this.shareStatus.set(this.t('share.clipboardUnavailable'));
    }
  }

  protected async sharePlan(): Promise<void> {
    const text = this.planSummaryText();
    if (text === '') {
      return;
    }
    if (!navigator.share) {
      this.shareStatus.set(this.t('share.nativeUnavailable'));
      return;
    }
    try {
      await navigator.share({ title: this.t('plan.title'), text });
      this.shareStatus.set(this.t('share.completed'));
    } catch (error: unknown) {
      this.shareStatus.set(
        error instanceof DOMException && error.name === 'AbortError'
          ? this.t('share.cancelled')
          : this.t('share.failed'),
      );
    }
  }

  protected async clearOfflineData(): Promise<void> {
    try {
      await this.offlineStorage.clearAll();
      this.setOfflineStatus('offline.clearSuccess');
      this.savedPlanStatusState.set(null);
      this.savedExplanationStatus.set('');
    } catch (error: unknown) {
      this.setOfflineStatus('offline.clearError', { detail: this.errorMessage(error) });
    }
  }

  protected previewBeacon(diagnostic: boolean): void {
    const stationId = this.selectedStationId();
    if (stationId === '' || this.beaconLoading()) {
      return;
    }
    const requestId = ++this.beaconRequestId;
    this.beaconLoading.set(true);
    this.beaconError.set('');
    this.beaconAnnouncement.set(this.t('beacon.loading'));
    this.api
      .simulateBeacon({
        stationId,
        steadyOnly: this.beaconSteadyOnly(),
        diagnostic,
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (requestId !== this.beaconRequestId) {
            return;
          }
          this.beaconLoading.set(false);
          this.beaconResult.set(result);
          this.beaconAnnouncement.set(
            diagnostic ? this.t('beacon.diagnosticReady') : this.t('beacon.previewReady'),
          );
        },
        error: (error: unknown) => {
          if (requestId !== this.beaconRequestId) {
            return;
          }
          this.beaconLoading.set(false);
          this.beaconError.set(this.errorMessage(error));
          this.beaconAnnouncement.set(this.t('beacon.error', { detail: this.beaconError() }));
        },
      });
  }

  protected beaconPatternLabel(pattern: BeaconPayload['pattern']): string {
    return this.t(`beacon.pattern.${pattern}`);
  }

  protected beaconColor(payload: BeaconPayload): string {
    return `rgb(${payload.rgb.red} ${payload.rgb.green} ${payload.rgb.blue})`;
  }

  private clearPlanOutput(): void {
    this.planRequestId += 1;
    this.explanationRequestId += 1;
    this.planLoading.set(false);
    this.planError.set('');
    this.planResult.set(null);
    this.planRequest.set(null);
    this.planAnnouncement.set('');
    this.explanationLoading.set(false);
    this.explanationError.set('');
    this.explanationResult.set(null);
    this.explanationAnnouncement.set('');
    this.savedPlanStatusState.set(null);
    this.savedExplanationStatus.set('');
    this.shareStatus.set('');
  }

  private loadStationData(stationId: string): Observable<DashboardData> {
    return this.api.getAirOutlook(stationId).pipe(
      switchMap((outlook) => {
        const anchor = Date.parse(outlook.anchor);
        if (Number.isNaN(anchor)) {
          throw new Error('The air-outlook response did not contain a valid IST anchor.');
        }
        const historyStart = formatISTHour(new Date(anchor - 30 * 24 * 60 * 60 * 1000));
        const forecastEnd = formatISTHour(new Date(anchor + 14 * 24 * 60 * 60 * 1000));
        return forkJoin({
          observations: this.api.getObservations(stationId, historyStart, outlook.anchor),
          forecasts: this.api.getForecast(stationId, outlook.anchor, forecastEnd),
        }).pipe(
          map(({ observations, forecasts }) => ({
            outlook,
            observations: observations.points,
            forecasts: forecasts.points,
            excludedFlaggedCount: observations.excludedFlaggedCount ?? 0,
          })),
        );
      }),
    );
  }

  private applyDashboardData(data: DashboardData): void {
    this.outlook.set(data.outlook);
    this.observations.set(data.observations);
    this.forecasts.set(data.forecasts);
    this.excludedFlaggedCount.set(data.excludedFlaggedCount);
    this.planDate.set(istDate(data.outlook.anchor));
    const sources = this.sources();
    const health = this.health();
    if (sources === null || health === null) {
      this.setOfflineStatus('offline.incomplete');
      return;
    }
    const observationStart = data.observations.reduce<string | null>(
      (earliest, point) =>
        earliest === null || point.intervalStart < earliest ? point.intervalStart : earliest,
      null,
    );
    const latestObservationEnd = data.observations.reduce<string | null>(
      (latest, point) =>
        latest === null || point.intervalEnd > latest ? point.intervalEnd : latest,
      null,
    );
    const forecastStart = data.forecasts.reduce<string | null>(
      (earliest, point) =>
        earliest === null || point.intervalStart < earliest ? point.intervalStart : earliest,
      null,
    );
    const forecastEnd = data.forecasts.reduce<string | null>(
      (latest, point) =>
        latest === null || point.intervalEnd > latest ? point.intervalEnd : latest,
      null,
    );
    const snapshot: OfflineSnapshot = {
      schemaVersion: 2,
      dataType: 'fixed_snapshot',
      stationId: data.outlook.stationId,
      sourceForecastSha256: sources.manifest.sources['forecasts']?.sha256 ?? '',
      modelName: this.aiStatus()?.model ?? null,
      coverage: {
        observationStart,
        latestObservationEnd,
        forecastStart,
        forecastEnd,
      },
      generatedAt: sources.manifest.forecastGeneratedAt,
      generationStatus: sources.manifest.forecastGenerationStatus,
      retrievedAt: new Date().toISOString(),
      excludedFlaggedCount: data.excludedFlaggedCount,
      stations: this.stations(),
      sources,
      health,
      aiStatus: this.aiStatus(),
      outlook: data.outlook,
      observations: data.observations,
      forecasts: data.forecasts,
    };
    void this.offlineStorage
      .saveSnapshot(snapshot)
      .then(() =>
        this.setOfflineStatus('offline.snapshotSaved', {
          station: snapshot.stationId,
          observationStart: snapshot.coverage.observationStart ?? '',
          observationEnd: snapshot.coverage.latestObservationEnd ?? '',
          forecastStart: snapshot.coverage.forecastStart ?? '',
          forecastEnd: snapshot.coverage.forecastEnd ?? '',
          generatedAt: snapshot.generatedAt ?? '',
          retrievedAt: snapshot.retrievedAt,
        }),
      )
      .catch((error: unknown) =>
        this.setOfflineStatus('offline.storageUnavailable', {
          detail: this.errorMessage(error),
        }),
      );
  }

  private async restoreLatestSnapshot(serviceError: string): Promise<void> {
    try {
      const snapshot = await this.offlineStorage.loadLatestSnapshot();
      if (snapshot === null) {
        this.dashboardLoading.set(false);
        this.dashboardError.set(
          this.t('offline.noSnapshot', { error: serviceError }),
        );
        return;
      }
      this.applyOfflineSnapshot(snapshot, serviceError);
    } catch (error: unknown) {
      this.dashboardLoading.set(false);
      this.dashboardError.set(
        this.t('offline.readSnapshotError', {
          error: serviceError,
          detail: this.errorMessage(error),
        }),
      );
    }
  }

  private async restoreSnapshot(stationId: string, serviceError: string): Promise<void> {
    try {
      const snapshot = await this.offlineStorage.loadSnapshot(stationId);
      if (this.selectedStationId() !== stationId) {
        return;
      }
      if (snapshot === null) {
        this.dashboardLoading.set(false);
        this.dashboardError.set(
          this.t('offline.noStationSnapshot', { error: serviceError, station: stationId }),
        );
        return;
      }
      this.applyOfflineSnapshot(snapshot, serviceError);
    } catch (error: unknown) {
      if (this.selectedStationId() !== stationId) {
        return;
      }
      this.dashboardLoading.set(false);
      this.dashboardError.set(
        this.t('offline.readSnapshotError', {
          error: serviceError,
          detail: this.errorMessage(error),
        }),
      );
    }
  }

  private applyOfflineSnapshot(snapshot: OfflineSnapshot, serviceError: string): void {
    this.stations.set(snapshot.stations);
    this.selectedStationId.set(snapshot.stationId);
    this.sources.set(snapshot.sources);
    this.health.set(snapshot.health);
    this.aiStatus.set(snapshot.aiStatus);
    this.capabilityStatuses.set(
      Object.fromEntries(
        Object.entries(snapshot.health.capabilities).map(([name, capability]) => [
          name,
          capability.status,
        ]),
      ),
    );
    if (snapshot.aiStatus !== null) {
      this.capabilityStatuses.update((capabilities) => ({
        ...capabilities,
        ai: snapshot.aiStatus?.status ?? 'unavailable',
      }));
    }
    this.outlook.set(snapshot.outlook);
    this.observations.set(snapshot.observations);
    this.forecasts.set(snapshot.forecasts);
    this.excludedFlaggedCount.set(snapshot.excludedFlaggedCount);
    this.planDate.set(istDate(snapshot.outlook.anchor));
    this.dashboardLoading.set(false);
    this.dashboardError.set('');
    this.clearPlanOutput();
    this.setOfflineStatus('offline.snapshotLoaded', {
      error: serviceError,
      station: snapshot.stationId,
      observationStart: snapshot.coverage.observationStart ?? '',
      observationEnd: snapshot.coverage.latestObservationEnd ?? '',
      forecastStart: snapshot.coverage.forecastStart ?? '',
      forecastEnd: snapshot.coverage.forecastEnd ?? '',
      generatedAt: snapshot.generatedAt ?? '',
      retrievedAt: snapshot.retrievedAt,
    });
  }

  private setOfflineStatus(
    key: string,
    values: Record<string, string | number> = {},
  ): void {
    this.offlineStatusMessage.set({ key, values });
  }

  private async persistPlan(
    request: ActivityPlanRequest,
    result: ActivityPlanResult,
    requestId: number,
  ): Promise<void> {
    try {
      await this.offlineStorage.savePlan(request, result);
      if (requestId === this.planRequestId) {
        this.savedPlanStatusState.set({ kind: 'saved' });
      }
    } catch (error: unknown) {
      if (requestId === this.planRequestId) {
        this.savedPlanStatusState.set({ kind: 'save-error', error });
      }
    }
  }

  private async restoreSavedPlan(request: ActivityPlanRequest, requestId: number): Promise<void> {
    if (requestId !== this.planRequestId) {
      return;
    }
    const sourceHash = this.sources()?.manifest.sources['forecasts']?.sha256 ?? '';
    try {
      const saved = await this.offlineStorage.loadPlan(request, sourceHash);
      if (requestId !== this.planRequestId) {
        return;
      }
      this.planLoading.set(false);
      if (saved === null) {
        this.planError.set(this.t('plan.noSavedPlan'));
        this.planAnnouncement.set(this.planError());
        return;
      }
      this.planResult.set(saved.result);
      this.planRequest.set(saved.request);
      this.planAnnouncement.set(
        this.t('plan.savedAt', { time: this.formatTime(saved.savedAt) }),
      );
      this.savedPlanStatusState.set({ kind: 'replayed', savedAt: saved.savedAt });
    } catch (error: unknown) {
      if (requestId !== this.planRequestId) {
        return;
      }
      this.planLoading.set(false);
      this.planError.set(
        this.t('offline.planReadError', { detail: this.errorMessage(error) }),
      );
      this.planAnnouncement.set(this.planError());
    }
  }

  private async persistExplanation(
    request: ActivityPlanRequest,
    language: LocaleCode,
    sourceForecastSha256: string,
    result: ExplanationResponse,
    requestId: number,
  ): Promise<void> {
    try {
      await this.offlineStorage.saveExplanation(request, language, sourceForecastSha256, result);
      if (requestId === this.explanationRequestId) {
        this.savedExplanationStatus.set(this.t('offline.explanationSaved'));
      }
    } catch (error: unknown) {
      if (requestId === this.explanationRequestId) {
        this.savedExplanationStatus.set(
          this.t('offline.explanationSaveError', { detail: this.errorMessage(error) }),
        );
      }
    }
  }

  private async restoreSavedExplanation(
    request: ActivityPlanRequest,
    requestId: number,
  ): Promise<void> {
    if (requestId !== this.explanationRequestId) {
      return;
    }
    const language = this.locale.current();
    const sourceHash = this.planResult()?.sourceForecastSha256 ?? '';
    try {
      const saved = await this.offlineStorage.loadExplanation(request, language, sourceHash);
      if (requestId !== this.explanationRequestId) {
        return;
      }
      this.explanationLoading.set(false);
      if (saved === null) {
        this.explanationError.set(this.t('plan.noSavedExplanation'));
        this.explanationAnnouncement.set(this.explanationError());
        return;
      }
      this.explanationResult.set(saved.result);
      this.explanationAnnouncement.set(
        this.t('plan.explanationSavedAt', { time: this.formatTime(saved.savedAt) }),
      );
      this.savedExplanationStatus.set(
        this.t('offline.explanationReplay', { time: this.formatTime(saved.savedAt) }),
      );
    } catch (error: unknown) {
      if (requestId !== this.explanationRequestId) {
        return;
      }
      this.explanationLoading.set(false);
      this.explanationError.set(
        this.t('offline.explanationReadError', { detail: this.errorMessage(error) }),
      );
      this.explanationAnnouncement.set(this.explanationError());
    }
  }

  protected planSummaryText(): string {
    const request = this.planRequest();
    const result = this.planResult();
    if (request === null || result === null) {
      return '';
    }
    const stationName =
      this.stations().find((station) => station.locationId === request.stationId)?.name ??
      this.t('station.generic', { stationId: request.stationId });
    const lines = [
      `Air Guardian · ${this.t('plan.title')} - ${stationName}`,
      `${this.t('form.activity')}: ${this.activityLabel(request.activityType)}`,
      `${this.t('plan.title')}: ${this.formatTime(result.recommendedPeriod.start)} – ${this.formatTime(result.recommendedPeriod.end)} IST`,
      `${this.t('plan.mean')} ${this.formatConcentration(result.recommendedPeriod.meanPm25)} · ${this.t('plan.intervals', { count: this.formatNumber(result.recommendedPeriod.coverageHours) })}`,
    ];
    if (result.comparison) {
      const preferredPeriod = result.comparison.preferredPeriod;
      lines.push(
        `${this.t('plan.comparison')}: ${this.formatTime(preferredPeriod.start)} – ${this.formatTime(preferredPeriod.end)} IST. ${this.t('plan.comparisonValue', {
          mean: this.formatConcentration(preferredPeriod.meanPm25),
          difference: this.formatSignedNumber(result.comparison.differencePm25),
        })}`,
      );
    }
    lines.push(this.t('plan.disclaimer'));
    return lines.join('\n');
  }

  private updateFormErrors(form: HTMLFormElement): void {
    const next: Record<string, string> = {};
    for (const element of Array.from(form.elements)) {
      if (element instanceof HTMLInputElement || element instanceof HTMLSelectElement) {
        const error = this.validationMessage(element);
        if (error !== '') {
          next[element.name] = error;
        }
      }
    }
    this.fieldErrors.set(next);
  }

  private validationMessage(control: HTMLInputElement | HTMLSelectElement): string {
    if (control.validity.valid) {
      return '';
    }
    if (control.validity.valueMissing) {
      return this.t('validation.required');
    }
    if (control.validity.stepMismatch) {
      return this.t('validation.wholeHour');
    }
    if (control.validity.rangeUnderflow || control.validity.rangeOverflow) {
      return this.t('validation.range');
    }
    return this.t('validation.invalid');
  }

  private errorMessage(error: unknown): string {
    if (error instanceof HttpErrorResponse) {
      if (error.status === 0) {
        return this.t('app.networkUnavailable');
      }

      const body = error.error as { message?: unknown; code?: unknown } | null;
      if (body && typeof body.code === 'string') {
        if (body.code === 'insufficient_coverage') {
          return this.t('app.insufficientCoverage');
        }
        if (body.code === 'invalid_language') {
          return this.t('app.invalidLanguage');
        }
        if (body.code === 'unknown_station') {
          return this.t('app.unknownStation');
        }
        if (body.code === 'explanation_unavailable') {
          return this.t('app.explanationUnavailable');
        }
        if (body.code === 'invalid_activity_request') {
          return this.t('app.planRequestFailed');
        }
      }
      if (body && typeof body.message === 'string') {
        return body.message;
      }
      return this.t('app.httpError', { status: error.status });
    }
    if (error instanceof Error) {
      return error.message;
    }
    return this.t('app.unexpected');
  }

  private canReplaySavedResult(error: unknown): error is HttpErrorResponse {
    return (
      error instanceof HttpErrorResponse &&
      (error.status === 0 || error.status === 502 || error.status === 503 || error.status === 504)
    );
  }

  private createChart(): {
    segments: ChartSegment[];
    ticks: ChartTick[];
    yMaximum: number;
    thresholds: { value: number; y: number }[];
    rangeLabel: string;
  } {
    const observations = this.observations()
      .filter((point) => !point.flagHigh1000)
      .map<PlotPoint>((point) => ({
        start: Date.parse(point.intervalStart),
        end: Date.parse(point.intervalEnd),
        time: Date.parse(point.intervalEnd),
        value: point.pm25,
        label: this.t('chart.pointDescription', {
          type: this.t('chart.observation'),
          concentration: this.formatConcentration(point.pm25),
          start: this.formatTime(point.intervalStart),
          end: this.formatTime(point.intervalEnd),
        }),
      }));
    const forecasts = this.forecasts().map<PlotPoint>((point) => ({
      start: Date.parse(point.intervalStart),
      end: Date.parse(point.intervalEnd),
      time: Date.parse(point.intervalEnd),
      value: point.pm25,
      label: this.t('chart.pointDescription', {
        type: this.t('chart.forecastRow'),
        concentration: this.formatConcentration(point.pm25),
        start: this.formatTime(point.intervalStart),
        end: this.formatTime(point.intervalEnd),
      }),
    }));
    const allPoints = [...observations, ...forecasts].filter(
      (point) =>
        Number.isFinite(point.start) &&
        Number.isFinite(point.end) &&
        Number.isFinite(point.time) &&
        Number.isFinite(point.value),
    );
    if (allPoints.length === 0) {
      return {
        segments: [],
        ticks: [],
        yMaximum: 120,
        thresholds: [],
        rangeLabel: this.t('chart.noDataRange'),
      };
    }

    const minTime = Math.min(...allPoints.map((point) => point.time));
    const maxTime = Math.max(...allPoints.map((point) => point.time));
    const minValue = 0;
    const maxValue = Math.max(...allPoints.map((point) => point.value));
    const yMaximum = Math.max(120, Math.ceil(maxValue / 30) * 30);
    const xPosition = (time: number) =>
      70 + ((time - minTime) / Math.max(1, maxTime - minTime)) * 900;
    const yPosition = (value: number) => 255 - ((value - minValue) / (yMaximum - minValue)) * 220;
    const segments = [
      ...this.buildSegments('observation', observations, xPosition, yPosition),
      ...this.buildSegments('forecast', forecasts, xPosition, yPosition),
    ];
    const ticks = Array.from({ length: 5 }, (_, index) => {
      const instant = minTime + ((maxTime - minTime) * index) / 4;
      return { x: xPosition(instant), label: this.formatShortTime(instant) };
    });
    const thresholds = [30, 60, 90]
      .filter((value) => value < yMaximum)
      .map((value) => ({ value, y: yPosition(value) }));
    return {
      segments,
      ticks,
      yMaximum,
      thresholds,
      rangeLabel: `${this.formatTime(new Date(minTime).toISOString())} – ${this.formatTime(new Date(maxTime).toISOString())}`,
    };
  }

  private buildSegments(
    kind: ChartSegment['kind'],
    points: PlotPoint[],
    xPosition: (time: number) => number,
    yPosition: (value: number) => number,
  ): ChartSegment[] {
    const sorted = [...points].sort((left, right) => left.start - right.start);
    const segments: ChartSegment[] = [];
    let current: PlotPoint[] = [];

    const saveCurrent = () => {
      if (current.length === 0) {
        return;
      }
      const chartPoints = current.map((point) => ({
        x: xPosition(point.time),
        y: yPosition(point.value),
        description: point.label,
      }));
      segments.push({
        id: `${kind}-${segments.length}`,
        kind,
        path: chartPoints
          .map(
            (point, index) =>
              `${index === 0 ? 'M' : 'L'} ${point.x.toFixed(2)} ${point.y.toFixed(2)}`,
          )
          .join(' '),
        points: chartPoints,
      });
      current = [];
    };

    for (const point of sorted) {
      if (current.length > 0 && current[current.length - 1].end !== point.start) {
        saveCurrent();
      }
      current.push(point);
    }
    saveCurrent();
    return segments;
  }

  private createChartRows(): {
    start: string;
    end: string;
    value: number;
    type: string;
    flag: string;
  }[] {
    const observations = this.observations().map((point) => ({
      start: point.intervalStart,
      end: point.intervalEnd,
      value: point.pm25,
      type: this.t('chart.observation'),
      flag: point.flagHigh1000
        ? this.t('chart.flagged')
        : this.t('chart.included'),
    }));
    const forecasts = this.forecasts().map((point) => ({
      start: point.intervalStart,
      end: point.intervalEnd,
      value: point.pm25,
      type: this.t('chart.forecastRow'),
      flag: this.t('chart.selectedForecast'),
    }));
    return [...observations, ...forecasts].sort(
      (left, right) => Date.parse(left.start) - Date.parse(right.start),
    );
  }
}

function formatISTHour(value: Date): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Kolkata',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(value);
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((value) => value.type === type)?.value ?? '';
  return `${part('year')}-${part('month')}-${part('day')}T${part('hour')}:${part('minute')}:${part('second')}+05:30`;
}

function istDate(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return '';
  }
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Kolkata',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(parsed);
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((value) => value.type === type)?.value ?? '';
  return `${part('year')}-${part('month')}-${part('day')}`;
}
