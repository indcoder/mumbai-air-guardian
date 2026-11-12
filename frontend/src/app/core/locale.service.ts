import { HttpClient } from '@angular/common/http';
import { DOCUMENT } from '@angular/common';
import { inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export type LocaleCode = 'en' | 'mr' | 'hi';

const localeCodes: readonly LocaleCode[] = ['en', 'mr', 'hi'];

function isLocaleCode(value: string): value is LocaleCode {
  return localeCodes.some((locale) => locale === value);
}

@Injectable({ providedIn: 'root' })
export class LocaleService {
  private readonly http = inject(HttpClient);
  private readonly document = inject(DOCUMENT);
  private readonly messages = signal<Record<string, string>>({});
  private englishKeys: string[] = [];
  private loadRequestId = 0;

  readonly current = signal<LocaleCode>('en');
  readonly error = signal('');

  async initialize(): Promise<void> {
    const messages = await this.loadMessages('en');
    this.englishKeys = Object.keys(messages).sort();
    this.applyLocale('en', messages);
  }

  async select(locale: string): Promise<boolean> {
    const requestId = ++this.loadRequestId;
    if (!isLocaleCode(locale)) {
      this.error.set(this.text('language.loadError'));
      return false;
    }
    if (locale === this.current()) {
      this.error.set('');
      return true;
    }

    this.error.set('');
    try {
      const messages = await this.loadMessages(locale);
      if (requestId !== this.loadRequestId) {
        return false;
      }
      this.applyLocale(locale, messages);
      return true;
    } catch {
      if (requestId === this.loadRequestId) {
        this.error.set(this.text('language.loadError'));
      }
      return false;
    }
  }

  text(key: string, values: Record<string, string | number> = {}): string {
    const template = this.messages()[key];
    if (template === undefined) {
      throw new Error(`Missing localized message: ${key}`);
    }
    return template.replace(/\{([^{}]+)\}/g, (placeholder, name: string) => {
      const value = values[name];
      if (value === undefined) {
        throw new Error(`Missing value "${name}" for localized message "${key}".`);
      }
      return String(value);
    });
  }

  localeTag(): string {
    switch (this.current()) {
      case 'mr':
        return 'mr-IN';
      case 'hi':
        return 'hi-IN';
      default:
        return 'en-IN';
    }
  }

  private async loadMessages(locale: LocaleCode): Promise<Record<string, string>> {
    const response = await firstValueFrom(
      this.http.get<unknown>(`/assets/i18n/${locale}.json`),
    );
    if (typeof response !== 'object' || response === null || Array.isArray(response)) {
      throw new Error(`Locale resource "${locale}" is not an object.`);
    }

    const messages: Record<string, string> = {};
    for (const [key, value] of Object.entries(response)) {
      if (typeof value !== 'string' || value.length === 0) {
        throw new Error(`Locale resource "${locale}" has an invalid value for "${key}".`);
      }
      messages[key] = value;
    }

    if (locale !== 'en') {
      const keys = Object.keys(messages).sort();
      if (
        keys.length !== this.englishKeys.length ||
        keys.some((key, index) => key !== this.englishKeys[index])
      ) {
        throw new Error(`Locale resource "${locale}" does not match the English key set.`);
      }
    }
    return messages;
  }

  private applyLocale(locale: LocaleCode, messages: Record<string, string>): void {
    this.messages.set(messages);
    this.current.set(locale);
    this.document.documentElement.lang = locale;
    this.document.title = messages['page.title'];
    this.error.set('');
  }
}
