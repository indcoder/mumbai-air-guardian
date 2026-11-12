import { Injectable } from '@angular/core';
import {
  ActivityPlanRequest,
  ActivityPlanResult,
  AirOutlook,
  AiStatusResponse,
  ExplanationResponse,
  ForecastPoint,
  HealthResponse,
  Observation,
  SourcesResponse,
  Station,
} from './air-guardian-api';

const databaseName = 'air-guardian-offline';
const databaseVersion = 2;
const snapshotStoreName = 'snapshots';
const planStoreName = 'plans';
const explanationStoreName = 'explanations';
const maxSavedResults = 128;
const schemaVersion = 2;

export interface OfflineSnapshot {
  schemaVersion: number;
  dataType: 'fixed_snapshot';
  stationId: string;
  sourceForecastSha256: string;
  modelName: string | null;
  coverage: {
    observationStart: string | null;
    latestObservationEnd: string | null;
    forecastStart: string | null;
    forecastEnd: string | null;
  };
  generatedAt: string | null;
  generationStatus: string;
  retrievedAt: string;
  excludedFlaggedCount: number;
  stations: Station[];
  sources: SourcesResponse;
  health: HealthResponse;
  aiStatus: AiStatusResponse | null;
  outlook: AirOutlook;
  observations: Observation[];
  forecasts: ForecastPoint[];
}

export interface SavedPlan {
  schemaVersion: number;
  key: string;
  savedAt: string;
  request: ActivityPlanRequest;
  result: ActivityPlanResult;
}

export interface SavedExplanation {
  schemaVersion: number;
  key: string;
  savedAt: string;
  request: ActivityPlanRequest;
  language: 'en' | 'mr' | 'hi';
  sourceForecastSha256: string;
  result: ExplanationResponse;
}

export function activityRequestKey(request: ActivityPlanRequest): string {
  return JSON.stringify([
    schemaVersion,
    request.stationId,
    request.activityType,
    request.date,
    request.earliestStart,
    request.latestEnd,
    request.durationMinutes,
    request.preferredStartTime ?? null,
  ]);
}

function isOfflineSnapshot(value: unknown): value is OfflineSnapshot {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const snapshot = value as Partial<OfflineSnapshot>;
  const forecastHash = snapshot.sources?.manifest?.sources['forecasts']?.sha256;
  const coverage = snapshot.coverage;
  const validTime = (time: string | null | undefined): boolean =>
    time === null || (typeof time === 'string' && Number.isFinite(Date.parse(time)));

  return (
    snapshot.schemaVersion === schemaVersion &&
    snapshot.dataType === 'fixed_snapshot' &&
    typeof snapshot.stationId === 'string' &&
    typeof snapshot.sourceForecastSha256 === 'string' &&
    /^[a-f0-9]{64}$/.test(snapshot.sourceForecastSha256) &&
    snapshot.sourceForecastSha256 === forecastHash &&
    snapshot.generatedAt === snapshot.sources?.manifest?.forecastGeneratedAt &&
    snapshot.generationStatus === snapshot.sources?.manifest?.forecastGenerationStatus &&
    typeof snapshot.retrievedAt === 'string' &&
    Number.isFinite(Date.parse(snapshot.retrievedAt)) &&
    Boolean(coverage) &&
    validTime(coverage?.observationStart) &&
    validTime(coverage?.latestObservationEnd) &&
    validTime(coverage?.forecastStart) &&
    validTime(coverage?.forecastEnd) &&
    Array.isArray(snapshot.stations) &&
    snapshot.stations.some((station) => station.locationId === snapshot.stationId) &&
    snapshot.outlook?.stationId === snapshot.stationId &&
    Array.isArray(snapshot.observations) &&
    Array.isArray(snapshot.forecasts)
  );
}

@Injectable({ providedIn: 'root' })
export class OfflineStorage {
  async saveSnapshot(snapshot: OfflineSnapshot): Promise<void> {
    if (!isOfflineSnapshot(snapshot)) {
      throw new Error('The offline snapshot is incomplete or incompatible.');
    }
    const database = await this.openDatabase();
    try {
      await this.write(database, snapshotStoreName, (store) => store.put(snapshot));
    } finally {
      database.close();
    }
  }

  async loadSnapshot(stationId: string): Promise<OfflineSnapshot | null> {
    const database = await this.openDatabase();
    try {
      const snapshot = await this.read<OfflineSnapshot | undefined>(
        database,
        snapshotStoreName,
        (store) => store.get(stationId),
      );
      return isOfflineSnapshot(snapshot) ? snapshot : null;
    } finally {
      database.close();
    }
  }

  async loadLatestSnapshot(): Promise<OfflineSnapshot | null> {
    const database = await this.openDatabase();
    try {
      const snapshots = await this.read<OfflineSnapshot[]>(
        database,
        snapshotStoreName,
        (store) => store.getAll() as IDBRequest<OfflineSnapshot[]>,
      );
      return (
        snapshots
          .filter(isOfflineSnapshot)
          .sort((left, right) => right.retrievedAt.localeCompare(left.retrievedAt))[0] ?? null
      );
    } finally {
      database.close();
    }
  }

  async savePlan(request: ActivityPlanRequest, result: ActivityPlanResult): Promise<void> {
    const record: SavedPlan = {
      schemaVersion,
      key: activityRequestKey(request),
      savedAt: new Date().toISOString(),
      request,
      result,
    };
    const database = await this.openDatabase();
    try {
      await this.writeBounded(database, planStoreName, record);
    } finally {
      database.close();
    }
  }

  async loadPlan(
    request: ActivityPlanRequest,
    sourceForecastSha256: string,
  ): Promise<SavedPlan | null> {
    const database = await this.openDatabase();
    try {
      const record = await this.read<SavedPlan | undefined>(
        database,
        planStoreName,
        (store) => store.get(activityRequestKey(request)),
      );
      return record?.schemaVersion === schemaVersion &&
        record.result.sourceForecastSha256 === sourceForecastSha256
        ? record
        : null;
    } finally {
      database.close();
    }
  }

  async saveExplanation(
    request: ActivityPlanRequest,
    language: 'en' | 'mr' | 'hi',
    sourceForecastSha256: string,
    result: ExplanationResponse,
  ): Promise<void> {
    const key = JSON.stringify([activityRequestKey(request), language]);
    const record: SavedExplanation = {
      schemaVersion,
      key,
      savedAt: new Date().toISOString(),
      request,
      language,
      sourceForecastSha256,
      result,
    };
    const database = await this.openDatabase();
    try {
      await this.writeBounded(database, explanationStoreName, record);
    } finally {
      database.close();
    }
  }

  async loadExplanation(
    request: ActivityPlanRequest,
    language: 'en' | 'mr' | 'hi',
    sourceForecastSha256: string,
  ): Promise<SavedExplanation | null> {
    const key = JSON.stringify([activityRequestKey(request), language]);
    const database = await this.openDatabase();
    try {
      const record = await this.read<SavedExplanation | undefined>(
        database,
        explanationStoreName,
        (store) => store.get(key),
      );
      return record?.schemaVersion === schemaVersion &&
        record.sourceForecastSha256 === sourceForecastSha256
        ? record
        : null;
    } finally {
      database.close();
    }
  }

  async clearAll(): Promise<void> {
    const database = await this.openDatabase();
    try {
      await new Promise<void>((resolve, reject) => {
        const transaction = database.transaction(
          [snapshotStoreName, planStoreName, explanationStoreName],
          'readwrite',
        );
        transaction.objectStore(snapshotStoreName).clear();
        transaction.objectStore(planStoreName).clear();
        transaction.objectStore(explanationStoreName).clear();
        transaction.oncomplete = () => resolve();
        transaction.onabort = () =>
          reject(transaction.error ?? new Error('Offline storage clear was aborted.'));
        transaction.onerror = () =>
          reject(transaction.error ?? new Error('Offline storage clear failed.'));
      });
    } finally {
      database.close();
    }
  }

  private openDatabase(): Promise<IDBDatabase> {
    const factory = globalThis.indexedDB;
    if (!factory) {
      return Promise.reject(new Error('IndexedDB is unavailable in this browser.'));
    }
    return new Promise((resolve, reject) => {
      const request = factory.open(databaseName, databaseVersion);
      request.onupgradeneeded = () => {
        const database = request.result;
        if (!database.objectStoreNames.contains(snapshotStoreName)) {
          database.createObjectStore(snapshotStoreName, { keyPath: 'stationId' });
        }
        if (!database.objectStoreNames.contains(planStoreName)) {
          database.createObjectStore(planStoreName, { keyPath: 'key' });
        }
        if (!database.objectStoreNames.contains(explanationStoreName)) {
          database.createObjectStore(explanationStoreName, { keyPath: 'key' });
        }
      };
      request.onsuccess = () => {
        request.result.onversionchange = () => request.result.close();
        resolve(request.result);
      };
      request.onerror = () => reject(request.error ?? new Error('Could not open offline storage.'));
      request.onblocked = () => reject(new Error('Offline storage upgrade is blocked by another tab.'));
    });
  }

  private read<T>(
    database: IDBDatabase,
    storeName: string,
    createRequest: (store: IDBObjectStore) => IDBRequest<T>,
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      const transaction = database.transaction(storeName, 'readonly');
      const request = createRequest(transaction.objectStore(storeName));
      let result: { value: T } | null = null;
      request.onsuccess = () => {
        result = { value: request.result };
      };
      request.onerror = () => reject(request.error ?? new Error('Could not read offline storage.'));
      transaction.oncomplete = () => {
        if (result === null) {
          reject(new Error('Offline storage read completed without a result.'));
          return;
        }
        resolve(result.value);
      };
      transaction.onabort = () =>
        reject(transaction.error ?? new Error('Offline storage read was aborted.'));
      transaction.onerror = () =>
        reject(transaction.error ?? new Error('Offline storage read failed.'));
    });
  }

  private write(
    database: IDBDatabase,
    storeName: string,
    createRequest: (store: IDBObjectStore) => IDBRequest,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const transaction = database.transaction(storeName, 'readwrite');
      const request = createRequest(transaction.objectStore(storeName));
      request.onerror = () => reject(request.error ?? new Error('Could not write offline storage.'));
      transaction.oncomplete = () => resolve();
      transaction.onabort = () =>
        reject(transaction.error ?? new Error('Offline storage write was aborted.'));
      transaction.onerror = () =>
        reject(transaction.error ?? new Error('Offline storage write failed.'));
    });
  }

  private writeBounded<T extends SavedPlan | SavedExplanation>(
    database: IDBDatabase,
    storeName: string,
    record: T,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const transaction = database.transaction(storeName, 'readwrite');
      const store = transaction.objectStore(storeName);
      store.put(record);
      const request = store.getAll() as IDBRequest<T[]>;
      request.onsuccess = () => {
        const records = request.result.sort((left, right) =>
          right.savedAt.localeCompare(left.savedAt),
        );
        for (const expired of records.slice(maxSavedResults)) {
          store.delete(expired.key);
        }
      };
      request.onerror = () => reject(request.error ?? new Error('Could not limit saved results.'));
      transaction.oncomplete = () => resolve();
      transaction.onabort = () =>
        reject(transaction.error ?? new Error('Offline storage update was aborted.'));
      transaction.onerror = () =>
        reject(transaction.error ?? new Error('Offline storage update failed.'));
    });
  }
}
