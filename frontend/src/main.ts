import { bootstrapApplication } from '@angular/platform-browser';
import { RUNTIME_CONFIG, parseRuntimeConfig } from './app/core/runtime-config';
import { appConfig } from './app/app.config';
import { App } from './app/app';
import { isDevMode } from '@angular/core';
import { provideServiceWorker } from '@angular/service-worker';

async function bootstrap(): Promise<void> {
  const response = await fetch('/runtime-config.json', { cache: 'no-store' });
  if (!response.ok) {
    throw new Error(`Runtime configuration request failed with HTTP ${response.status}.`);
  }
  const config = parseRuntimeConfig(await response.json());
  await bootstrapApplication(App, {
    ...appConfig,
    providers: [
      ...appConfig.providers,
      { provide: RUNTIME_CONFIG, useValue: config },
      provideServiceWorker('ngsw-worker.js', {
        enabled: !isDevMode(),
        registrationStrategy: 'registerWhenStable:30000',
      }),
    ],
  });
}

bootstrap().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : 'Application startup failed.';
  const startupError = document.getElementById('startup-error');
  if (startupError) {
    startupError.textContent = `Air Guardian could not start: ${message}`;
    startupError.hidden = false;
  }
  console.error(error);
});
