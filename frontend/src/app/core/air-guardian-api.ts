import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { RUNTIME_CONFIG } from './runtime-config';

export interface Station {
  locationId: string;
  name: string;
  sensorId: string;
  latitude: number;
  longitude: number;
  timezone: string;
}

export interface Observation {
  stationId: string;
  sensorId: string;
  intervalStart: string;
  intervalEnd: string;
  sourceTimestamp: string;
  availableAtSourceMtime: string;
  pm25: number;
  mean: number;
  count: number;
  stationName: string;
  seriesId: string;
  flagHigh1000: boolean;
  sourceKind: 'observed';
}

export interface ForecastPoint {
  id: string;
  stationId: string;
  sensorId: string;
  intervalStart: string;
  intervalEnd: string;
  pm25: number;
  sourceKind: 'forecast';
  sourceRow: number;
}

export interface RiskBand {
  name: 'low' | 'elevated' | 'high' | 'very_high';
  lower: number;
  lowerInclusive: boolean;
  upper: number | null;
  upperInclusive: boolean;
  color: string;
  icon: string;
  beaconPattern: string;
}

export interface AirOutlook {
  stationId: string;
  anchor: string;
  latestObservation?: Observation;
  observationBand?: RiskBand;
  peak: { valuePm25: number; band: RiskBand };
  lowest: { valuePm25: number; band: RiskBand };
  trajectory: {
    status: 'unavailable' | 'improving' | 'stable' | 'worsening';
    slopeUgM3PerHour?: number;
    pointsUsed: number;
    partialCoverage: boolean;
    lookaheadHours: number;
  };
  forecastGenerationTime: string;
}

export interface SourceRecord {
  kind: 'observed' | 'forecast' | 'reference';
  path: string;
  sha256: string;
  rows: number;
}

export interface SourcesResponse {
  snapshotLabel: string;
  latestObservationLabel: string;
  forecastGenerationTime: string;
  manifest: {
    timezone: string;
    forecastGeneratedAt: string | null;
    forecastGenerationStatus: string;
    selectedForecastScalingAlreadyApplied: number;
    sources: Record<string, SourceRecord>;
    climateVerseCatalogue: {
      doi: string;
      version: string;
      stage: string;
      discoveredVia: string;
      catalogueRole: string;
      dataProvider: string;
    };
  };
}

export interface HealthResponse {
  status: string;
  capabilities: Record<string, { status: string; reason?: string }>;
}

export interface AiStatusResponse {
  status: string;
  reason?: string;
  model: string;
  languages: string[];
  generatedLanguages: string[];
}

export interface ExplanationResponse {
  mode: 'gemma' | 'template';
  reasonCode?: string;
  evidenceHash: string;
  explanation: {
    summary: string;
    keySignals: string[];
    recommendedAction: string;
    limitations: string;
    language: string;
  };
}

export interface ActivityPlanRequest {
  stationId: string;
  activityType: string;
  date: string;
  earliestStart: string;
  latestEnd: string;
  durationMinutes: number;
  preferredStartTime?: string;
}

export interface ActivityPeriod {
  start: string;
  end: string;
  meanPm25: number;
  coverageHours: number;
}

export interface ActivityPlanResult {
  stationId: string;
  activityType: string;
  recommendedPeriod: ActivityPeriod;
  comparison?: {
    preferredPeriod: ActivityPeriod;
    differencePm25: number;
  };
  resolutionMinutes: number;
  sourceForecastSha256: string;
  forecastGenerationTime: string;
}

export interface BeaconPayload {
  version: number;
  stationId: string;
  pm25: number;
  band: RiskBand['name'];
  trajectory: AirOutlook['trajectory']['status'];
  peakHour: string;
  rgb: { red: number; green: number; blue: number };
  pattern: 'steady' | 'single-pulse' | 'double-pulse' | 'triple-pulse' | 'diagnostic';
  diagnostic: boolean;
  steadyOnly: boolean;
}

export interface BeaconStatusResponse {
  status: 'simulated';
  mode: 'simulation';
  reason: string;
  lastPayload: BeaconPayload | null;
}

export interface TimedSeriesResponse<T> {
  stationId: string;
  from: string;
  to: string;
  points: T[];
  source: SourceRecord;
  excludedFlaggedCount?: number;
  exclusionReason?: string;
  generatedAt?: string | null;
  generationStatus?: string;
}

@Injectable({ providedIn: 'root' })
export class AirGuardianApi {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = inject(RUNTIME_CONFIG).apiBaseUrl;

  getStations(): Observable<{ stations: Station[] }> {
    return this.http.get<{ stations: Station[] }>(`${this.baseUrl}/api/stations`);
  }

  getSources(): Observable<SourcesResponse> {
    return this.http.get<SourcesResponse>(`${this.baseUrl}/api/sources`);
  }

  getHealth(): Observable<HealthResponse> {
    return this.http.get<HealthResponse>(`${this.baseUrl}/api/health`);
  }

  getAiStatus(): Observable<AiStatusResponse> {
    return this.http.get<AiStatusResponse>(`${this.baseUrl}/api/ai/status`);
  }

  getAirOutlook(stationId: string): Observable<AirOutlook> {
    return this.http.get<AirOutlook>(
      `${this.baseUrl}/api/stations/${encodeURIComponent(stationId)}/air-outlook`,
    );
  }

  getObservations(
    stationId: string,
    from: string,
    to: string,
  ): Observable<TimedSeriesResponse<Observation>> {
    return this.http.get<TimedSeriesResponse<Observation>>(
      `${this.baseUrl}/api/stations/${encodeURIComponent(stationId)}/observations`,
      { params: new HttpParams().set('from', from).set('to', to) },
    );
  }

  getForecast(
    stationId: string,
    from: string,
    to: string,
  ): Observable<TimedSeriesResponse<ForecastPoint>> {
    return this.http.get<TimedSeriesResponse<ForecastPoint>>(
      `${this.baseUrl}/api/stations/${encodeURIComponent(stationId)}/forecast`,
      { params: new HttpParams().set('from', from).set('to', to) },
    );
  }

  getContext(stationId: string): Observable<{ status: string; reason: string }> {
    return this.http.get<{ status: string; reason: string }>(
      `${this.baseUrl}/api/stations/${encodeURIComponent(stationId)}/context`,
    );
  }

  createActivityPlan(request: ActivityPlanRequest): Observable<ActivityPlanResult> {
    return this.http.post<ActivityPlanResult>(`${this.baseUrl}/api/activity-plan`, request);
  }

  createExplanation(
    request: ActivityPlanRequest,
    language: 'en' | 'mr' | 'hi',
  ): Observable<ExplanationResponse> {
    return this.http.post<ExplanationResponse>(`${this.baseUrl}/api/explain`, request, {
      params: new HttpParams().set('lang', language),
    });
  }

  simulateBeacon(request: {
    stationId: string;
    steadyOnly: boolean;
    diagnostic: boolean;
  }): Observable<BeaconStatusResponse> {
    return this.http.post<BeaconStatusResponse>(`${this.baseUrl}/api/beacon`, request);
  }
}
