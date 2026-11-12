import { HttpClient } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import english from '../../assets/i18n/en.json';
import hindi from '../../assets/i18n/hi.json';
import marathi from '../../assets/i18n/mr.json';
import { LocaleService } from './locale.service';

describe('LocaleService', () => {
  const resources: Record<string, unknown> = {
    '/assets/i18n/en.json': english,
    '/assets/i18n/mr.json': marathi,
    '/assets/i18n/hi.json': hindi,
  };
  const http = {
    get: vi.fn((url: string) => of(resources[url])),
  };

  beforeEach(() => {
    http.get.mockImplementation((url: string) => of(resources[url]));
    TestBed.configureTestingModule({
      providers: [{ provide: HttpClient, useValue: http }],
    });
  });

  it('keeps locale resource keys aligned', () => {
    const englishKeys = Object.keys(english).sort();
    expect(Object.keys(marathi).sort()).toEqual(englishKeys);
    expect(Object.keys(hindi).sort()).toEqual(englishKeys);
  });

  it('updates the document language, title, and interpolated messages', async () => {
    const locale = TestBed.inject(LocaleService);
    await locale.initialize();

    expect(document.documentElement.lang).toBe('en');
    expect(document.title).toBe(english['page.title']);

    expect(await locale.select('mr')).toBe(true);
    expect(locale.current()).toBe('mr');
    expect(locale.localeTag()).toBe('mr-IN');
    expect(document.documentElement.lang).toBe('mr');
    expect(document.title).toBe(marathi['page.title']);
    expect(locale.text('station.summary', { stationId: '11611', sensorId: 'sensor-1' })).toBe(
      'केंद्र 11611 · सेन्सर sensor-1',
    );
  });

  it('retains the active locale and reports a load error when a resource fails', async () => {
    const locale = TestBed.inject(LocaleService);
    await locale.initialize();
    http.get.mockImplementationOnce(() => throwError(() => new Error('resource unavailable')));

    expect(await locale.select('hi')).toBe(false);
    expect(locale.current()).toBe('en');
    expect(locale.error()).toBe(english['language.loadError']);
    expect(document.documentElement.lang).toBe('en');
  });

  it('rejects missing interpolation values rather than displaying incomplete text', async () => {
    const locale = TestBed.inject(LocaleService);
    await locale.initialize();

    expect(() => locale.text('station.summary', { stationId: '11611' })).toThrow(
      'Missing value "sensorId" for localized message "station.summary".',
    );
  });
});
