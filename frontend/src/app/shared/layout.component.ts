import { Component, signal } from '@angular/core';
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
      <button class="hamburger" (click)="sidebarOpen.set(!sidebarOpen())">
        <mat-icon>{{ sidebarOpen() ? 'close' : 'menu' }}</mat-icon>
      </button>
      @if (sidebarOpen()) {
        <div class="sidebar-backdrop" (click)="sidebarOpen.set(false)"></div>
      }
      <div class="sidebar" [class.open]="sidebarOpen()">
        <div class="sidebar-header">
          <h2>MateRAG</h2>
        </div>
        <nav class="sidebar-nav">
          <a routerLink="/chat" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>chat</mat-icon><span>Chat</span></a>
          <a routerLink="/math" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>calculate</mat-icon><span>Matematica</span></a>
          <a routerLink="/tutor" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>school</mat-icon><span>Tutor</span></a>
          <a routerLink="/documents" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>folder</mat-icon><span>Documentos</span></a>
          <a routerLink="/bdvectorial" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>database</mat-icon><span>BD Vectorial</span></a>
          <a routerLink="/history" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>history</mat-icon><span>Historial</span></a>
          <a routerLink="/assessment" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>quiz</mat-icon><span>Evaluaciones</span></a>
          <a routerLink="/my-progress" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>trending_up</mat-icon><span>Mi Progreso</span></a>
          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <a routerLink="/dashboard" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>dashboard</mat-icon><span>Panel</span></a>
            <a routerLink="/teacher" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>analytics</mat-icon><span>Panel Profesor</span></a>
          }
          @if (auth.hasRole('ADMIN')) {
            <a routerLink="/settings" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>settings</mat-icon><span>Configuracion</span></a>
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
        <div class="sidebar-copyright">
          <a href="https://softgroup.com.ar" target="_blank" rel="noopener">softgroup.com.ar</a>
          &copy; 2026
        </div>
      </div>
      <div class="content">
        <router-outlet />
      </div>
    </div>
  `,
  styles: [`
    .layout { display: flex; height: 100vh; background: var(--bg); color: var(--text); }
    .hamburger { display: none; position: fixed; top: 0.75rem; left: 0.75rem; z-index: 1100; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; color: var(--text); width: 40px; height: 40px; cursor: pointer; align-items: center; justify-content: center; }
    .hamburger mat-icon { font-size: 22px; }
    .sidebar-backdrop { display: none; }
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
    .sidebar-copyright { padding: 0.6rem 1rem; border-top: 1px solid var(--border); text-align: center; font-size: 0.7rem; color: var(--text-secondary); }
    .sidebar-copyright a { color: var(--text-secondary); text-decoration: none; }
    .sidebar-copyright a:hover { color: var(--accent); }
    .content { flex: 1; overflow-y: auto; }

    @media (max-width: 1023px) {
      .hamburger { display: flex; }
      .sidebar { position: fixed; top: 0; left: -260px; bottom: 0; z-index: 1200; transition: left 0.25s ease; width: 260px; }
      .sidebar.open { left: 0; }
      .sidebar-backdrop { display: block; position: fixed; inset: 0; z-index: 1100; background: rgba(0,0,0,0.5); }
      .content { padding-top: 3.25rem; }
    }
  `]
})
export class LayoutComponent {
  sidebarOpen = signal(false);
  constructor(public auth: AuthService, public themeService: ThemeService) {}
}
