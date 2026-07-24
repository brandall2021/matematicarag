import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="container">
      <h1>Dashboard Admin</h1>
      <div class="stats-grid">
        <div class="stat-card"><h3>{{ stats().totalUsers }}</h3><p>Usuarios</p></div>
        <div class="stat-card"><h3>{{ stats().totalSessions }}</h3><p>Sesiones</p></div>
        <div class="stat-card"><h3>{{ stats().totalMessages }}</h3><p>Mensajes</p></div>
        <div class="stat-card"><h3>{{ stats().totalDocuments }}</h3><p>Documentos</p></div>
      </div>
    </div>
  `,
  styles: [`
    .container { padding: 2rem; max-width: 800px; margin: 0 auto; background: #1a1a2e; color: white; min-height: 100vh; }
    h1 { color: #e2b714; }
    .stats-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 1rem; margin-top: 2rem; }
    .stat-card { background: #16213e; padding: 2rem; border-radius: 12px; text-align: center; }
    .stat-card h3 { font-size: 2.5rem; color: #e2b714; margin: 0; }
    .stat-card p { color: #888; margin-top: 0.5rem; }
  `]
})
export class DashboardComponent implements OnInit {
  stats = signal<any>({ totalUsers: 0, totalSessions: 0, totalMessages: 0, totalDocuments: 0 });
  constructor(private api: ApiService) {}
  ngOnInit() { this.api.getAdminStats().subscribe(s => this.stats.set(s)); }
}
