import { bootstrapApplication } from '@angular/platform-browser';
import { AppComponent } from './app/app.component';
import { appConfig } from './app/app.config';

bootstrapApplication(AppComponent, appConfig)
  .then(() => {
    if (location.protocol === 'https:' || location.hostname === 'localhost') {
      if ('serviceWorker' in navigator) {
        navigator.serviceWorker.register('ngsw-worker.js').catch(err => {
          console.warn('Service worker registration failed', err);
        });
      }
    }
  })
  .catch(err => console.error(err));
