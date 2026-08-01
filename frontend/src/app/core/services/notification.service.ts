import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, interval, tap } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface AppNotification {
  id: string;
  type: string;
  title: string;
  message: string;
  link: string;
  read: boolean;
  created_at: string;
}

@Injectable({ providedIn: 'root' })
export class NotificationService {
  private baseUrl = environment.apiUrl + '/api';
  private unreadSignal = signal(0);
  readonly unread = computed(() => this.unreadSignal());
  open = false;

  constructor(private http: HttpClient) {
    interval(60000).subscribe(() => this.refreshUnread());
  }

  getNotifications(): Observable<AppNotification[]> {
    return this.http.get<AppNotification[]>(`${this.baseUrl}/notifications?limit=20`);
  }

  refreshUnread(): void {
    this.http.get<{ count: number }>(`${this.baseUrl}/notifications/unread-count`)
      .subscribe({ next: r => this.unreadSignal.set(r.count), error: () => {} });
  }

  markRead(id: string): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/notifications/${id}/read`, {})
      .pipe(tap(() => this.unreadSignal.update(n => Math.max(0, n - 1))));
  }

  markAllRead(): Observable<void> {
    return this.http.put<void>(`${this.baseUrl}/notifications/read-all`, {})
      .pipe(tap(() => this.unreadSignal.set(0)));
  }
}
