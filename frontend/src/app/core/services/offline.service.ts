import { Injectable, signal, computed } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class OfflineService {
  private offlineSignal = signal(typeof navigator !== 'undefined' ? !navigator.onLine : false);
  readonly isOffline = computed(() => this.offlineSignal());

  constructor() {
    if (typeof window === 'undefined') return;
    window.addEventListener('online', () => {
      this.offlineSignal.set(false);
      this.retry();
    });
    window.addEventListener('offline', () => this.offlineSignal.set(true));
  }

  markOffline(): void {
    this.offlineSignal.set(true);
  }

  retry(): void {
    this.offlineSignal.set(false);
    window.location.reload();
  }
}
