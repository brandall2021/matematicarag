import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { MatIconModule } from '@angular/material/icon';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule, MatIconModule],
  template: `
    <div class="container">
      <h1><mat-icon>dashboard</mat-icon> Panel de Administración</h1>
      <p class="period-caption">Resumen general de la plataforma</p>
      @if (loading()) {
        <div class="loading-state"><mat-icon class="spin">sync</mat-icon> Cargando estadísticas...</div>
      } @else {
        <div class="stats-grid">
          <div class="stat-card">
            <mat-icon class="stat-icon">people</mat-icon>
            <h3>{{ stats().totalUsers }}</h3>
            <p>Usuarios registrados</p>
          </div>
          <div class="stat-card">
            <mat-icon class="stat-icon">forum</mat-icon>
            <h3>{{ stats().totalSessions }}</h3>
            <p>Sesiones de chat</p>
          </div>
          <div class="stat-card">
            <mat-icon class="stat-icon">chat</mat-icon>
            <h3>{{ stats().totalMessages }}</h3>
            <p>Mensajes enviados</p>
          </div>
          <div class="stat-card">
            <mat-icon class="stat-icon">description</mat-icon>
            <h3>{{ stats().totalDocuments }}</h3>
            <p>Documentos indexados</p>
          </div>
        </div>
        @if (stats().totalDocuments === 0) {
          <div class="empty-docs">
            <mat-icon>description</mat-icon>
            <p>No hay documentos indexados todavía. Subí material desde la sección Documentos.</p>
          </div>
        }
      }
    </div>
  `,
  styles: [`
    .container { padding: 1.5rem; max-width: 900px; margin: 0 auto; }
    @media (min-width: 768px) { .container { padding: 2rem; } }
    h1 { color: var(--accent); display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1.5rem; font-family: 'Newsreader', serif; }
    h1 mat-icon { font-size: 28px; width: 28px; height: 28px; }
    .loading-state { display: flex; align-items: center; gap: 0.5rem; justify-content: center; padding: 3rem; color: var(--text-secondary); }
    .spin { animation: spin 1s linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
    .stats-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 1rem; }
    @media (max-width: 767px) { .stats-grid { grid-template-columns: 1fr; } }
    .stat-card { background: var(--surface); padding: 1.5rem; border-radius: 12px; text-align: center; border: 1px solid var(--border); transition: border-color 0.2s; }
    .stat-card:hover { border-color: var(--accent); }
    .stat-icon { font-size: 2rem; width: 2rem; height: 2rem; color: var(--text-secondary); margin-bottom: 0.5rem; }
    .stat-card h3 { font-size: 2.5rem; color: var(--accent); margin: 0; font-weight: 700; }
    .stat-card p { color: var(--text-secondary); margin-top: 0.25rem; font-size: 0.85rem; }
    .period-caption { color: var(--text-secondary); font-size: 0.85rem; margin: -1rem 0 1.5rem 0; }
    .empty-docs { display: flex; align-items: center; gap: 0.5rem; margin-top: 1rem; padding: 1rem; background: var(--surface); border: 1px solid var(--border); border-radius: 12px; color: var(--text-secondary); font-size: 0.85rem; }
    .empty-docs mat-icon { color: var(--accent); }
  `]
})
export class DashboardComponent implements OnInit {
  stats = signal<any>({ totalUsers: 0, totalSessions: 0, totalMessages: 0, totalDocuments: 0 });
  loading = signal(true);
  constructor(private api: ApiService) {}
  ngOnInit() {
    this.api.getAdminStats().subscribe({
      next: (s) => { this.stats.set(s); this.loading.set(false); },
      error: () => this.loading.set(false)
    });
  }
}
