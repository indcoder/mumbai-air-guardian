import 'fake-indexeddb/auto';
import { beforeEach, describe, expect, it } from 'vitest';
import { ActivityPlanRequest, ActivityPlanResult, ExplanationResponse } from './air-guardian-api';
import { OfflineStorage, OfflineSnapshot } from './offline-storage';

async function countRecords(storeName: string): Promise<number> {
  const database = await new Promise<IDBDatabase>((resolve, reject) => {
    const request = indexedDB.open('air-guardian-offline', 2);
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
  try {
    return await new Promise<number>((resolve, reject) => {
      const transaction = database.transaction(storeName, 'readonly');
      const request = transaction.objectStore(storeName).count();
      let count = 0;
      request.onsuccess = () => {
        count = request.result;
      };
      transaction.oncomplete = () => resolve(count);
      transaction.onerror = () => reject(transaction.error);
      transaction.onabort = () => reject(transaction.error);
    });
  } finally {
    database.close();
  }
}

describe('OfflineStorage', () => {
  let storage: OfflineStorage;

  const request: ActivityPlanRequest = {
    stationId: '11611',
    activityType: 'outdoor exercise',
    date: '2026-11-12',
    earliestStart: '06:00',
    latestEnd: '20:00',
    durationMinutes: 120,
  };
  const plan: ActivityPlanResult = {
    stationId: '11611',
    activityType: 'outdoor exercise',
    recommendedPeriod: {
      start: '2026-11-12T15:00:00+05:30',
      end: '2026-11-12T17:00:00+05:30',
      meanPm25: 52.58,
      coverageHours: 2,
    },
    resolutionMinutes: 60,
    sourceForecastSha256: 'a'.repeat(64),
    forecastGenerationTime: 'unavailable',
  };
  const explanation: ExplanationResponse = {
    mode: 'template',
    reasonCode: 'generation_timeout',
    evidenceHash: 'b'.repeat(64),
    explanation: {
      summary: 'A deterministic explanation.',
      keySignals: ['Complete forecast coverage.'],
      recommendedAction: 'Use as a relative comparison.',
      limitations: 'Fixed snapshot.',
      language: 'en',
    },
  };
  const snapshot: OfflineSnapshot = {
    schemaVersion: 2,
    dataType: 'fixed_snapshot',
    stationId: '11611',
    sourceForecastSha256: 'a'.repeat(64),
    modelName: null,
    coverage: {
      observationStart: null,
      latestObservationEnd: null,
      forecastStart: null,
      forecastEnd: null,
    },
    generatedAt: null,
    generationStatus: 'unavailable',
    retrievedAt: '2026-11-12T12:00:00.000Z',
    excludedFlaggedCount: 0,
    stations: [
      {
        locationId: '11611',
        name: 'Malad',
        sensorId: 'sensor-1',
        latitude: 19.18,
        longitude: 72.84,
        timezone: 'Asia/Kolkata',
      },
    ],
    sources: {
      snapshotLabel: 'Challenge forecast snapshot',
      latestObservationLabel: 'Latest observation in supplied dataset',
      forecastGenerationTime: 'unavailable',
      manifest: {
        timezone: 'Asia/Kolkata',
        forecastGeneratedAt: null,
        forecastGenerationStatus: 'unavailable',
        selectedForecastScalingAlreadyApplied: 1.1,
        sources: {
          forecasts: {
            kind: 'forecast',
            path: 'submission.csv',
            sha256: 'a'.repeat(64),
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
    },
    health: { status: 'ok', capabilities: {} },
    aiStatus: null,
    outlook: {
      stationId: '11611',
      anchor: '2026-11-10T09:00:00+05:30',
      peak: {
        valuePm25: 60,
        band: {
          name: 'high',
          lower: 60,
          lowerInclusive: true,
          upper: 90,
          upperInclusive: false,
          color: '#a44',
          icon: 'warning',
          beaconPattern: 'steady',
        },
      },
      lowest: {
        valuePm25: 20,
        band: {
          name: 'low',
          lower: 0,
          lowerInclusive: true,
          upper: 30,
          upperInclusive: true,
          color: '#2d6a4f',
          icon: 'check',
          beaconPattern: 'steady',
        },
      },
      trajectory: {
        status: 'unavailable',
        pointsUsed: 0,
        partialCoverage: true,
        lookaheadHours: 6,
      },
      forecastGenerationTime: 'unavailable',
    },
    observations: [],
    forecasts: [],
  };

  beforeEach(async () => {
    storage = new OfflineStorage();
    await storage.clearAll();
  });

  it('round-trips only the current compatible dashboard snapshot', async () => {
    await storage.saveSnapshot(snapshot);

    await expect(storage.loadSnapshot('11611')).resolves.toEqual(snapshot);
    await expect(storage.loadLatestSnapshot()).resolves.toEqual(snapshot);
    await expect(storage.loadSnapshot('unknown')).resolves.toBeNull();
  });

  it('replays a saved plan only for identical inputs and the same forecast source', async () => {
    await storage.savePlan(request, plan);

    await expect(storage.loadPlan(request, plan.sourceForecastSha256)).resolves.toMatchObject({
      request,
      result: plan,
    });
    await expect(storage.loadPlan(request, 'c'.repeat(64))).resolves.toBeNull();
    await expect(
      storage.loadPlan({ ...request, preferredStartTime: '15:00' }, plan.sourceForecastSha256),
    ).resolves.toBeNull();
  });

  it('keeps explanations separated by language and source snapshot', async () => {
    await storage.saveExplanation(request, 'en', plan.sourceForecastSha256, explanation);

    await expect(
      storage.loadExplanation(request, 'en', plan.sourceForecastSha256),
    ).resolves.toMatchObject({ result: explanation });
    await expect(
      storage.loadExplanation(request, 'mr', plan.sourceForecastSha256),
    ).resolves.toBeNull();
    await expect(storage.loadExplanation(request, 'en', 'c'.repeat(64))).resolves.toBeNull();
  });

  it('removes all saved snapshots and results on request', async () => {
    await storage.saveSnapshot(snapshot);
    await storage.savePlan(request, plan);
    await storage.saveExplanation(request, 'en', plan.sourceForecastSha256, explanation);

    await storage.clearAll();

    await expect(storage.loadLatestSnapshot()).resolves.toBeNull();
    await expect(storage.loadPlan(request, plan.sourceForecastSha256)).resolves.toBeNull();
    await expect(
      storage.loadExplanation(request, 'en', plan.sourceForecastSha256),
    ).resolves.toBeNull();
  });

  it('bounds saved plans and explanations to the most recent 128 entries', async () => {
    for (let index = 0; index < 130; index += 1) {
      const uniqueRequest = { ...request, activityType: `activity-${index}` };
      await storage.savePlan(uniqueRequest, plan);
      await storage.saveExplanation(
        uniqueRequest,
        'en',
        plan.sourceForecastSha256,
        explanation,
      );
    }

    await expect(countRecords('plans')).resolves.toBe(128);
    await expect(countRecords('explanations')).resolves.toBe(128);
  });
});
