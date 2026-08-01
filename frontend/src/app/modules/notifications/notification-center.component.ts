import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { Router } from '@angular/router';
import { NotificationService, AppNotification } from '../../core/services/notification.service';

@Component({
  selector: 'app-notification-center',
  standalone: true,
  imports: [CommonModule, MatIconModule],
  template: `
    <div class="notif-wrap" (click)="stop($event)">
      <button mat-icon-button class="notif-bell" (click)="toggle()" [attr.aria-label]="'Notificaciones'">
        <mat-icon>notifications</mat-icon>
        @if (service.unread() > 0) {
          <span class="notif-badge">{{ service.unread() > 9 ? '9+' : service.unread() }}</span>
        }
      </button>

      @if (open()) {
        <div class="notif-panel">
          <div class="notif-header">
            <span>Notificaciones</span>
            <button mat-button class="notif-mark-all" (click)="markAllRead()">Marcar todas como leídas</button>
          </div>
          @if (items(); as list) {
            @if (list.length === 0) {
              <div class="notif-empty">Sin notificaciones.</div>
            } @else {
              <div class="notif-list">
                @for (n of list; track n.id) {
                  <div class="notif-item" [class.unread]="!n.read" (click)="openNotification(n)">
                    <div class="notif-title">{{ n.title }}</div>
                    <div class="notif-message">{{ n.message }}</div>
                    <div class="notif-time">{{ n.created_at | date:'dd/MM/yyyy HH:mm' }}</div>
                  </div>
                }
              </div>
            }
          } @else {
            <div class="notif-empty">Cargando...</div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    .notif-wrap { position: relative; }
    .notif-bell { width: 32px; height: 32px; color: var(--text-secondary); position: relative; }
    .notif-bell:hover { color: var(--accent); }
    .notif-badge {
      position: absolute; top: 2px; right: 2px;
      background: var(--warn, #e53935); color: #fff;
      border-radius: 10px; font-size: 0.6rem; font-weight: 700;
      min-width: 16px; height: 16px; display: flex; align-items: center; justify-content: center;
      padding: 0 3px;
    }
    .notif-panel {
      position: absolute; top: 2.5rem; right: 0; z-index: 1400;
      width: 320px; max-height: 420px; overflow-y: auto;
      background: var(--surface-elevated); border: 1px solid var(--border);
      border-radius: var(--radius-md); box-shadow: var(--shadow-lg);
      display: flex; flex-direction: column;
    }
    .notif-header {
      display: flex; justify-content: space-between; align-items: center;
      padding: 0.6rem 0.75rem; border-bottom: 1px solid var(--border);
      font-weight: 600; font-size: 0.85rem; position: sticky; top: 0;
      background: var(--surface-elevated);
    }
    .notif-mark-all { font-size: 0.7rem; color: var(--accent-text); cursor: pointer; background: none; border: none; }
    .notif-empty { padding: 1.5rem; text-align: center; color: var(--text-secondary); font-size: 0.85rem; }
    .notif-item { padding: 0.6rem 0.75rem; border-bottom: 1px solid var(--border-light); cursor: pointer; }
    .notif-item:hover { background: var(--accent-muted); }
    .notif-item.unread { border-left: 3px solid var(--accent); }
    .notif-title { font-weight: 600; font-size: 0.85rem; }
    .notif-message { font-size: 0.78rem; color: var(--text-secondary); margin-top: 0.15rem; }
    .notif-time { font-size: 0.68rem; color: var(--text-tertiary); margin-top: 0.2rem; }
  `]
})
export class NotificationCenterComponent implements OnInit {
  open = signal(false);
  items = signal<AppNotification[] | null>(null);

  constructor(public service: NotificationService, private router: Router) {}

  ngOnInit(): void {
    this.service.refreshUnread();
  }

  toggle(): void {
    this.open.update(o => !o);
    if (this.open()) this.load();
  }

  load(): void {
    this.service.getNotifications().subscribe({ next: n => this.items.set(n), error: () => this.items.set([]) });
  }

  markAllRead(): void {
    this.service.markAllRead().subscribe({ next: () => this.load() });
  }

  openNotification(n: AppNotification): void {
    this.service.markRead(n.id).subscribe();
    this.open.set(false);
    if (n.link) this.router.navigateByUrl(n.link);
  }

  stop(e: Event): void { e.stopPropagation(); }
}
