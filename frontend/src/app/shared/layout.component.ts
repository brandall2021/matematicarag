import { Component, signal } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../core/services/auth.service';
import { ThemeService } from '../core/services/theme.service';
import { OfflineService } from '../core/services/offline.service';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive, MatButtonModule, MatIconModule, MatTooltipModule],
  template: `
    <div class="layout">
      @if (offline.isOffline()) {
        <div class="offline-banner" role="status">
          <mat-icon>cloud_off</mat-icon>
          <span>Estás sin conexión. Algunas funciones pueden no estar disponibles.</span>
          <button mat-button class="offline-retry" (click)="offline.retry()">Volver a intentar</button>
        </div>
      }
      <button class="hamburger" (click)="sidebarOpen.set(!sidebarOpen())" aria-label="Alternar menú">
        <mat-icon>{{ sidebarOpen() ? 'close' : 'menu' }}</mat-icon>
      </button>
      @if (sidebarOpen()) {
        <div class="sidebar-backdrop" (click)="sidebarOpen.set(false)"></div>
      }
      <aside class="sidebar" [class.open]="sidebarOpen()" role="navigation" aria-label="Main navigation">
        <div class="sidebar-header">
          <mat-icon class="sidebar-logo-icon">calculate</mat-icon>
          <span class="sidebar-logo-text">Mate<span class="logo-accent">RAG</span></span>
        </div>

        <nav class="sidebar-nav">
          <div class="nav-section-label">Principal</div>
          <a routerLink="/chat" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)" matTooltip="Consultá documentos y recibí respuestas con fuentes"><mat-icon>chat</mat-icon><span>Chat</span></a>
          <a routerLink="/agente" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)" matTooltip="Analiza tu problema y guía tu aprendizaje paso a paso"><mat-icon>smart_toy</mat-icon><span>Agente</span></a>
          <a routerLink="/math" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)" matTooltip="Operaciones simbólicas: derivar, integrar, simplificar"><mat-icon>calculate</mat-icon><span>Matemática</span></a>
          <a routerLink="/tutor" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)" matTooltip="Ejercicios prácticos, verificación y ayuda paso a paso"><mat-icon>school</mat-icon><span>Tutor</span></a>

          <div class="nav-section-label">Contenido</div>
          <a routerLink="/documents" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>folder</mat-icon><span>{{ auth.hasRole('ADMIN', 'TEACHER') ? 'Documentos' : 'Material' }}</span></a>
          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <a routerLink="/bdvectorial" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>database</mat-icon><span>BD Vectorial</span></a>
          }
          <a routerLink="/history" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>history</mat-icon><span>Historial</span></a>

          <div class="nav-section-label">Aprendizaje</div>
          <a routerLink="/assessment" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>quiz</mat-icon><span>Evaluaciones</span></a>
          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <a routerLink="/analytics" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>analytics</mat-icon><span>Analíticas</span></a>
          }
          <a routerLink="/my-progress" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>trending_up</mat-icon><span>Mi Progreso</span></a>
          <a routerLink="/aprendizaje" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>insights</mat-icon><span>Aprendizaje</span></a>
          <a routerLink="/logros" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>emoji_events</mat-icon><span>Logros</span></a>

          @if (auth.hasRole('ADMIN', 'TEACHER')) {
            <div class="nav-section-label">Gestión</div>
            <a routerLink="/dashboard" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>dashboard</mat-icon><span>Panel</span></a>
            <a routerLink="/teacher" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>assignment</mat-icon><span>Panel Profesor</span></a>
          }
          @if (auth.hasRole('ADMIN')) {
            <a routerLink="/settings" routerLinkActive="active" class="nav-item" (click)="sidebarOpen.set(false)"><mat-icon>settings</mat-icon><span>Configuración</span></a>
          }
        </nav>

        <div class="sidebar-footer">
          <div class="user-info">
            <div class="user-avatar">{{ auth.currentUser()?.name?.charAt(0)?.toUpperCase() || 'U' }}</div>
            <span class="user-name">{{ auth.currentUser()?.name || 'Usuario' }}</span>
          </div>
          <div class="footer-actions">
            <button mat-icon-button (click)="themeService.toggle()" class="theme-btn" aria-label="Cambiar tema" matTooltip="Cambiar tema">
              <mat-icon>{{ themeService.currentTheme() === 'dark' ? 'light_mode' : 'dark_mode' }}</mat-icon>
            </button>
            <button mat-icon-button (click)="auth.logout()" aria-label="Cerrar sesión" matTooltip="Cerrar sesión">
              <mat-icon>logout</mat-icon>
            </button>
          </div>
        </div>

        <div class="sidebar-copyright">
          <a href="https://softgroup.com.ar" target="_blank" rel="noopener">SoftGroup</a>
          <span>&middot;</span>
          <span>&copy; 2026</span>
        </div>
      </aside>
      <main class="content">
        <router-outlet />
      </main>
    </div>
  `,
  styles: [`
    .layout { display: flex; height: 100vh; background: var(--bg); color: var(--text); }
    .hamburger { display: none; position: fixed; top: 0.75rem; left: 0.75rem; z-index: 1100; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-md); color: var(--text); width: 40px; height: 40px; cursor: pointer; align-items: center; justify-content: center; }
    .hamburger:hover { background: var(--surface-elevated); border-color: var(--accent); }
    .hamburger mat-icon { font-size: 22px; }
    .sidebar-backdrop { display: none; }

    .offline-banner {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      z-index: 1300;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 1rem;
      background: var(--surface-elevated);
      border-bottom: 1px solid var(--border);
      color: var(--text);
      font-size: 0.8rem;
      box-shadow: var(--shadow-md);
    }
    .offline-banner mat-icon { color: var(--danger); flex-shrink: 0; }
    .offline-retry { margin-left: auto; }

    .sidebar {
      width: 220px;
      background: var(--surface);
      display: flex;
      flex-direction: column;
      border-right: 1px solid var(--border);
      flex-shrink: 0;
      transition: background 0.3s;
    }

    .sidebar-header {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: var(--space-lg) var(--space-md) var(--space-md);
      border-bottom: 1px solid var(--border-light);
    }
    .sidebar-logo-icon { color: var(--accent); font-size: 28px; width: 28px; height: 28px; }
    .sidebar-logo-text { font-family: var(--font-serif); font-size: 1.15rem; font-weight: 600; color: var(--text); letter-spacing: -0.02em; }
    .logo-accent { color: var(--accent); }

    .sidebar-nav { flex: 1; padding: var(--space-sm); overflow-y: auto; }
    .nav-section-label {
      padding: var(--space-md) var(--space-sm) var(--space-xs);
      font-size: 0.65rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--text-tertiary);
    }
    .nav-item {
      display: flex;
      align-items: center;
      gap: var(--space-sm);
      padding: 0.5rem var(--space-sm);
      border-radius: var(--radius-sm);
      color: var(--text-secondary);
      text-decoration: none;
      font-size: 0.85rem;
      cursor: pointer;
      margin-bottom: 1px;
      transition: all 0.12s ease;
      position: relative;
    }
    .nav-item:hover { background: var(--accent-muted); color: var(--text); }
    .nav-item.active {
      background: var(--accent-muted);
      color: var(--accent-text);
      font-weight: 500;
    }
    .nav-item.active::before {
      content: '';
      position: absolute;
      left: -8px;
      top: 50%;
      transform: translateY(-50%);
      width: 3px;
      height: 60%;
      background: var(--accent);
      border-radius: 0 2px 2px 0;
    }
    .nav-item mat-icon { font-size: 18px; width: 18px; height: 18px; flex-shrink: 0; }

    .sidebar-footer {
      padding: var(--space-sm) var(--space-md);
      border-top: 1px solid var(--border-light);
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .user-info { display: flex; align-items: center; gap: var(--space-sm); min-width: 0; }
    .user-avatar {
      width: 28px; height: 28px; border-radius: 50%;
      background: var(--accent-muted); color: var(--accent-text);
      display: flex; align-items: center; justify-content: center;
      font-size: 0.75rem; font-weight: 700; flex-shrink: 0;
    }
    .user-name { color: var(--text-secondary); font-size: 0.8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .footer-actions { display: flex; gap: 2px; }
    .footer-actions button { width: 32px; height: 32px; }

    .sidebar-copyright {
      padding: 0.5rem var(--space-md);
      display: flex;
      align-items: center;
      gap: 0.35rem;
      justify-content: center;
      font-size: 0.65rem;
      color: var(--text-tertiary);
      border-top: 1px solid var(--border-light);
    }
    .sidebar-copyright a { color: var(--text-tertiary); text-decoration: none; transition: color 0.15s; }
    .sidebar-copyright a:hover { color: var(--accent); }

    .content { flex: 1; overflow-y: auto; }

    @media (max-width: 1023px) {
      .hamburger { display: flex; }
      .sidebar { position: fixed; top: 0; left: -240px; bottom: 0; z-index: 1200; transition: left 0.25s ease; width: 240px; box-shadow: var(--shadow-lg); }
      .sidebar.open { left: 0; }
      .sidebar-backdrop { display: block; position: fixed; inset: 0; z-index: 1100; background: var(--overlay); }
      .content { padding-top: 3.5rem; }
    }
  `]
})
export class LayoutComponent {
  sidebarOpen = signal(false);
  constructor(public auth: AuthService, public themeService: ThemeService, public offline: OfflineService) {}
}
