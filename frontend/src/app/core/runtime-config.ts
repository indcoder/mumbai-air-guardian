import { InjectionToken } from '@angular/core';

export interface RuntimeConfig {
  apiBaseUrl: string;
}

export const RUNTIME_CONFIG = new InjectionToken<RuntimeConfig>('RUNTIME_CONFIG');

export function parseRuntimeConfig(value: unknown): RuntimeConfig {
  if (typeof value !== 'object' || value === null || !('apiBaseUrl' in value)) {
    throw new Error('Runtime configuration must include apiBaseUrl.');
  }

  const rawUrl = (value as { apiBaseUrl?: unknown }).apiBaseUrl;
  if (typeof rawUrl !== 'string' || rawUrl.length === 0) {
    throw new Error('Runtime apiBaseUrl must be a non-empty URL.');
  }

  let url: URL;
  try {
    url = new URL(rawUrl);
  } catch {
    throw new Error('Runtime apiBaseUrl must be an absolute URL.');
  }

  const allowedHosts = new Set(['localhost', '127.0.0.1', '[::1]']);
  if (
    !['http:', 'https:'].includes(url.protocol) ||
    !allowedHosts.has(url.hostname.toLowerCase()) ||
    url.username !== '' ||
    url.password !== '' ||
    url.pathname !== '/' ||
    url.search !== '' ||
    url.hash !== ''
  ) {
    throw new Error(
      'Runtime apiBaseUrl must target a local HTTP(S) host without credentials or a path.',
    );
  }

  return { apiBaseUrl: url.origin };
}
