import { Component, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { ApiService } from '../../core/services/api.service';

@Component({
  selector: 'app-history',
  standalone: true,
  imports: [CommonModule, MatIconModule],
  template: `
    <div class="container">
      <h1>Historial</h1>
      <div class="search-box">
        <mat-icon>search</mat-icon>
        <input #searchInput
               placeholder="Buscar por título..."
               (input)="query.set(searchInput.value)"
               aria-label="Buscar en el historial">
      </div>

      @if (filteredHistory().length === 0) {
        <div class="empty-state">
          @if (query()) {
            <p>No se encontraron conversaciones para "{{ query() }}".</p>
          } @else {
            <p>Aún no hay historial.</p>
          }
        </div>
      }

      @for (entry of filteredHistory(); track entry.id) {
        <div class="history-item">
          <div class="history-icon"><mat-icon>chat</mat-icon></div>
          <div class="history-body">
            <h3>{{ entry.title }}</h3>
            <p>{{ entry.messages }} mensajes · {{ entry.createdAt | date:'dd/MM/yyyy, HH:mm' }}</p>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .container { padding: 1.5rem; max-width: 800px; margin: 0 auto; background: var(--bg); color: var(--text); }
    @media (min-width: 768px) { .container { padding: 2rem; } }
    h1 { color: var(--accent); margin-bottom: 1.5rem; font-family: 'Newsreader', serif; }
    .search-box { display: flex; align-items: center; gap: 0.5rem; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 0.5rem 0.75rem; margin-bottom: 1.5rem; }
    .search-box mat-icon { color: var(--text-tertiary); }
    .search-box input { flex: 1; border: none; outline: none; background: transparent; color: var(--text); font-size: 0.9rem; }
    .history-item { display: flex; gap: 0.75rem; align-items: flex-start; background: var(--surface); padding: 1rem; border-radius: 8px; margin-bottom: 0.5rem; border: 1px solid var(--border); }
    .history-icon mat-icon { color: var(--accent); margin-top: 2px; }
    .history-body h3 { margin: 0 0 0.25rem 0; color: var(--text); font-size: 0.95rem; }
    .history-body p { margin: 0; color: var(--text-secondary); font-size: 0.8rem; }
    .empty-state { text-align: center; margin-top: 3rem; color: var(--text-secondary); }
  `]
})
export class HistoryComponent implements OnInit {
  history = signal<any[]>([]);
  query = signal('');

  filteredHistory = computed(() => {
    const q = this.query().trim().toLowerCase();
    if (!q) return this.history();
    return this.history().filter(h => (h.title || '').toLowerCase().includes(q));
  });

  constructor(private api: ApiService) {}
  ngOnInit() { this.api.getHistory().subscribe(h => this.history.set(h)); }
}
