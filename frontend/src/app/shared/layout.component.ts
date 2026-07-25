import { Component } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../core/services/auth.service';
import { ThemeService } from '../core/services/theme.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive, MatButtonModule, MatIconModule],
  template: `
    <div class="layout">
      <div class="sidebar">
        <div class="sidebar-header">
          <h2>MateRAG</h2>
        </div>
        <nav class="sidebar-nav">
          <a routerLink="/chat" routerLinkActive="active" class="nav-item"><mat-icon>chat</mat-icon><span>Chat</span></a>
          <a routerLink="/math" routerLinkActive="active" class="nav-item"><mat-icon>calculate</mat-icon><span>Matematica</span></a>
          <a routerLink="/documents" routerLinkActive="active" class="nav-item"><mat-icon>folder</mat-icon><span>Documentos</span></a>
          <a routerLink="/history" routerLinkActive="active" class="nav-item"><mat-icon>history</mat-icon><span>Historial</span></a>
          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <a routerLink="/dashboard" routerLinkActive="active" class="nav-item"><mat-icon>dashboard</mat-icon><span>Panel</span></a>
          }
          @if (auth.hasRole('ADMIN')) {
            <a routerLink="/settings" routerLinkActive="active" class="nav-item"><mat-icon>settings</mat-icon><span>Configuracion</span></a>
          }
        </nav>
        <div class="sidebar-footer">
          <span class="user-name">{{ auth.currentUser()?.name }}</span>
          <div class="footer-actions">
            <button mat-icon-button (click)="themeService.toggle()" class="theme-btn">
              <mat-icon>{{ themeService.currentTheme() === 'dark' ? 'light_mode' : 'dark_mode' }}</mat-icon>
            </button>
            <button mat-icon-button (click)="auth.logout()">
              <mat-icon>logout</mat-icon>
            </button>
          </div>
        </div>
      </div>
      <div class="content">
        <router-outlet />
      </div>
    </div>
  `,
  styles: [`
    .layout { display: flex; height: 100vh; background: var(--bg); color: var(--text); }
    .sidebar { width: 240px; background: var(--surface); display: flex; flex-direction: column; border-right: 1px solid var(--border); flex-shrink: 0; }
    .sidebar-header { padding: 1rem 1.25rem; border-bottom: 1px solid var(--border); }
    .sidebar-header h2 { color: var(--accent); font-size: 1.1rem; margin: 0; font-weight: 700; }
    .sidebar-nav { flex: 1; padding: 0.5rem; overflow-y: auto; }
    .nav-item { display: flex; align-items: center; gap: 0.75rem; padding: 0.6rem 0.75rem; border-radius: 8px; color: var(--text-secondary); text-decoration: none; font-size: 0.9rem; cursor: pointer; margin-bottom: 2px; }
    .nav-item:hover { background: var(--border); color: var(--text); }
    .nav-item.active { background: var(--accent); color: var(--bg); }
    .nav-item mat-icon { font-size: 20px; width: 20px; height: 20px; }
    .sidebar-footer { padding: 0.75rem 1rem; border-top: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center; }
    .user-name { color: var(--text-secondary); font-size: 0.8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .footer-actions { display: flex; gap: 0; }
    .footer-actions button { width: 32px; height: 32px; }
    .footer-actions button mat-icon { font-size: 18px; width: 18px; height: 18px; color: var(--text-secondary); }
    .theme-btn:hover mat-icon, .footer-actions button:hover mat-icon { color: var(--accent); }
    .content { flex: 1; overflow-y: auto; }
  `]
})
export class LayoutComponent {
  constructor(public auth: AuthService, public themeService: ThemeService) {}
}
