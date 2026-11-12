import { parseRuntimeConfig } from './runtime-config';

describe('parseRuntimeConfig', () => {
  it('accepts a local API origin and normalizes its trailing slash', () => {
    expect(parseRuntimeConfig({ apiBaseUrl: 'http://localhost:8080/' })).toEqual({
      apiBaseUrl: 'http://localhost:8080',
    });
  });

  it('rejects remote hosts, credentials, and URL paths', () => {
    expect(() => parseRuntimeConfig({ apiBaseUrl: 'https://example.com' })).toThrow();
    expect(() => parseRuntimeConfig({ apiBaseUrl: 'http://user:pass@localhost:8080' })).toThrow();
    expect(() => parseRuntimeConfig({ apiBaseUrl: 'http://localhost:8080/api' })).toThrow();
  });

  it('rejects missing or non-string API base URLs', () => {
    expect(() => parseRuntimeConfig(null)).toThrow();
    expect(() => parseRuntimeConfig({ apiBaseUrl: 8080 })).toThrow();
  });
});
